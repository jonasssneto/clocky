package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"clocky/internal/oauth"
	"clocky/internal/settings"
)

const (
	spotifyAccountsURL = "https://accounts.spotify.com"
	spotifyAPIURL      = "https://api.spotify.com/v1"
	spotifyScopes      = "user-read-currently-playing"
	spotifyBodyLimit   = 1 << 20
)

var ErrSpotifyAuthorizationRequired = errors.New("spotify authorization required")

type spotifyHTTPError struct {
	operation string
	status    int
	message   string
}

func (e *spotifyHTTPError) Error() string {
	return fmt.Sprintf("%s: HTTP %d: %s", e.operation, e.status, e.message)
}

type Track struct {
	Title      string
	Artist     string
	Playing    bool
	ElapsedSec int
	TotalSec   int
	CoverURL   string
}

type spotifyConfig struct {
	clientID    string
	redirectURI string
	tokenFile   string
}

type spotifyToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type SpotifyClient struct {
	mu       sync.Mutex
	config   spotifyConfig
	http     *http.Client
	accounts string
	api      string
	token    spotifyToken
	verifier string
}

var _ oauth.Integration = (*SpotifyClient)(nil)

func NewSpotifyClient(snapshot settings.Snapshot) (*SpotifyClient, error) {
	config := spotifyConfig{
		clientID:    strings.TrimSpace(snapshot.SpotifyClientID),
		redirectURI: strings.TrimSpace(snapshot.SpotifyRedirectURI),
	}
	if config.clientID == "" {
		return nil, errors.New("SPOTIFY_CLIENT_ID is required")
	}
	if config.redirectURI == "" {
		config.redirectURI = "http://127.0.0.1:8888/callback"
	}
	var err error
	config.tokenFile, err = defaultSpotifyTokenFile()
	if err != nil {
		return nil, err
	}
	if err := validateSpotifyRedirect(config.redirectURI); err != nil {
		return nil, err
	}

	client := &SpotifyClient{
		config:   config,
		http:     &http.Client{Timeout: 10 * time.Second},
		accounts: spotifyAccountsURL,
		api:      spotifyAPIURL,
	}
	if err := client.loadToken(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load Spotify token: %w", err)
	}
	return client, nil
}

func defaultSpotifyTokenFile() (string, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(configDirectory, "clocky", "spotify-token.json"), nil
}

func validateSpotifyRedirect(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid SPOTIFY_REDIRECT_URI: %w", err)
	}
	if parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" {
		return errors.New("SPOTIFY_REDIRECT_URI must use http://127.0.0.1:PORT/path")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("SPOTIFY_REDIRECT_URI must not include credentials, a query, or a fragment")
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return errors.New("SPOTIFY_REDIRECT_URI must include a callback path")
	}
	return nil
}

func (c *SpotifyClient) ID() string {
	return "spotify"
}

func (c *SpotifyClient) Name() string {
	return "Spotify"
}

func (c *SpotifyClient) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token.RefreshToken != ""
}

func (c *SpotifyClient) ConfigurationPageURL() string {
	redirect, _ := url.Parse(c.config.redirectURI)
	redirect.Path = "/"
	redirect.RawPath = ""
	redirect.RawQuery = ""
	redirect.Fragment = ""
	return redirect.String()
}

func (c *SpotifyClient) BeginAuthorization() (oauth.Authorization, error) {
	state, err := randomURLToken(24)
	if err != nil {
		return oauth.Authorization{}, fmt.Errorf("generate Spotify authorization state: %w", err)
	}
	verifier, err := randomURLToken(64)
	if err != nil {
		return oauth.Authorization{}, fmt.Errorf("generate Spotify PKCE verifier: %w", err)
	}
	challengeHash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])
	params := url.Values{
		"client_id":             {c.config.clientID},
		"response_type":         {"code"},
		"redirect_uri":          {c.config.redirectURI},
		"scope":                 {spotifyScopes},
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {challenge},
		"show_dialog":           {"true"},
	}
	redirect, err := url.Parse(c.config.redirectURI)
	if err != nil {
		return oauth.Authorization{}, err
	}
	c.mu.Lock()
	c.verifier = verifier
	c.mu.Unlock()
	return oauth.Authorization{
		URL:          c.accounts + "/authorize?" + params.Encode(),
		State:        state,
		CallbackPath: redirect.Path,
	}, nil
}

func randomURLToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (c *SpotifyClient) CompleteAuthorization(ctx context.Context, code string) error {
	c.mu.Lock()
	verifier := c.verifier
	c.verifier = ""
	c.mu.Unlock()
	if verifier == "" {
		return errors.New("Spotify authorization session expired; start again") //nolint:staticcheck // Spotify is a proper noun.
	}
	return c.exchangeAuthorizationCode(ctx, code, verifier)
}

func (c *SpotifyClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = spotifyToken{}
	c.verifier = ""
	err := os.Remove(c.config.tokenFile)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove Spotify token: %w", err)
	}
	return nil
}

func (c *SpotifyClient) CurrentTrack(ctx context.Context) (Track, error) {
	accessToken, err := c.validAccessToken(ctx)
	if err != nil {
		return Track{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.api+"/me/player/currently-playing", nil)
	if err != nil {
		return Track{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := c.http.Do(request)
	if err != nil {
		return Track{}, fmt.Errorf("get current Spotify track: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return Track{Title: "Nothing playing", Artist: "Open Spotify to start playback"}, nil
	}
	if response.StatusCode == http.StatusUnauthorized {
		c.clearToken()
		return Track{}, ErrSpotifyAuthorizationRequired
	}
	if response.StatusCode != http.StatusOK {
		return Track{}, spotifyResponseError("get current Spotify track", response)
	}

	var playback struct {
		IsPlaying bool `json:"is_playing"`
		Progress  int  `json:"progress_ms"`
		Item      struct {
			Name     string `json:"name"`
			Duration int    `json:"duration_ms"`
			Artists  []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Images []struct {
					URL string `json:"url"`
				} `json:"images"`
			} `json:"album"`
		} `json:"item"`
	}
	if err := decodeSpotifyJSON(response.Body, &playback); err != nil {
		return Track{}, fmt.Errorf("decode current Spotify track: %w", err)
	}
	if playback.Item.Name == "" {
		return Track{Title: "Nothing playing", Artist: "Open Spotify to start playback"}, nil
	}
	artists := make([]string, 0, len(playback.Item.Artists))
	for _, artist := range playback.Item.Artists {
		artists = append(artists, artist.Name)
	}
	coverURL := ""
	if len(playback.Item.Album.Images) > 0 {
		coverURL = playback.Item.Album.Images[0].URL
	}
	return Track{
		Title:      playback.Item.Name,
		Artist:     strings.Join(artists, ", "),
		Playing:    playback.IsPlaying,
		ElapsedSec: playback.Progress / 1000,
		TotalSec:   playback.Item.Duration / 1000,
		CoverURL:   coverURL,
	}, nil
}

func (c *SpotifyClient) validAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token.RefreshToken == "" {
		return "", ErrSpotifyAuthorizationRequired
	}
	if c.token.AccessToken != "" && time.Now().Add(30*time.Second).Before(c.token.ExpiresAt) {
		return c.token.AccessToken, nil
	}
	if err := c.refreshToken(ctx); err != nil {
		var responseError *spotifyHTTPError
		if errors.As(err, &responseError) && (responseError.status == http.StatusBadRequest || responseError.status == http.StatusUnauthorized) {
			c.token = spotifyToken{}
			_ = os.Remove(c.config.tokenFile)
			return "", errors.Join(ErrSpotifyAuthorizationRequired, fmt.Errorf("refresh Spotify token: %w", err))
		}
		return "", fmt.Errorf("refresh Spotify token: %w", err)
	}
	return c.token.AccessToken, nil
}

func (c *SpotifyClient) exchangeAuthorizationCode(ctx context.Context, code, verifier string) error {
	values := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.config.redirectURI},
		"code_verifier": {verifier},
	}
	token, err := c.requestToken(ctx, values)
	if err != nil {
		return fmt.Errorf("exchange Spotify authorization code: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
	return c.saveToken()
}

func (c *SpotifyClient) refreshToken(ctx context.Context) error {
	refreshToken := c.token.RefreshToken
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	token, err := c.requestToken(ctx, values)
	if err != nil {
		return err
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	c.token = token
	return c.saveToken()
}

func (c *SpotifyClient) requestToken(ctx context.Context, values url.Values) (spotifyToken, error) {
	values.Set("client_id", c.config.clientID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.accounts+"/api/token", strings.NewReader(values.Encode()))
	if err != nil {
		return spotifyToken{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.http.Do(request)
	if err != nil {
		return spotifyToken{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return spotifyToken{}, spotifyResponseError("request Spotify token", response)
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := decodeSpotifyJSON(response.Body, &payload); err != nil {
		return spotifyToken{}, err
	}
	if payload.AccessToken == "" {
		return spotifyToken{}, errors.New("Spotify token response did not include an access token") //nolint:staticcheck // Spotify is a proper noun.
	}
	return spotifyToken{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		Scope:        payload.Scope,
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
	}, nil
}

func spotifyResponseError(operation string, response *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(response.StatusCode)
	}
	return &spotifyHTTPError{operation: operation, status: response.StatusCode, message: message}
}

func decodeSpotifyJSON(reader io.Reader, target any) error {
	return json.NewDecoder(io.LimitReader(reader, spotifyBodyLimit)).Decode(target)
}

func (c *SpotifyClient) loadToken() error {
	file, err := os.Open(c.config.tokenFile)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(io.LimitReader(file, spotifyBodyLimit)).Decode(&c.token)
}

func (c *SpotifyClient) saveToken() error {
	contents, err := json.MarshalIndent(c.token, "", "  ")
	if err != nil {
		return err
	}
	directory := filepath.Dir(c.config.tokenFile)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".spotify-token-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, c.config.tokenFile)
}

func (c *SpotifyClient) clearToken() {
	_ = c.Disconnect()
}
