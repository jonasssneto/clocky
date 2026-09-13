package data

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateSpotifyRedirect(t *testing.T) {
	t.Parallel()

	valid := "http://127.0.0.1:8888/callback"
	if err := validateSpotifyRedirect(valid); err != nil {
		t.Fatalf("validateSpotifyRedirect(%q): %v", valid, err)
	}

	invalid := []string{
		"http://localhost:8888/callback",
		"https://127.0.0.1:8888/callback",
		"http://127.0.0.1/callback",
		"http://127.0.0.1:8888/",
		"http://127.0.0.1:8888/callback?code=value",
	}
	for _, redirect := range invalid {
		if err := validateSpotifyRedirect(redirect); err == nil {
			t.Errorf("validateSpotifyRedirect(%q) unexpectedly succeeded", redirect)
		}
	}
}

func TestBeginAuthorizationUsesPKCE(t *testing.T) {
	t.Parallel()

	client := testSpotifyClient(t)
	auth, err := client.BeginAuthorization()
	if err != nil {
		t.Fatal(err)
	}
	if auth.CallbackPath != "/callback" {
		t.Errorf("CallbackPath = %q", auth.CallbackPath)
	}
	parsed, err := url.Parse(auth.URL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-id" {
		t.Errorf("client_id = %q", query.Get("client_id"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Errorf("code_challenge_method = %q", query.Get("code_challenge_method"))
	}
	hash := sha256.Sum256([]byte(client.verifier))
	wantChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	if query.Get("code_challenge") != wantChallenge {
		t.Error("code_challenge does not match the verifier")
	}
	if query.Get("state") == "" || query.Get("state") != auth.State {
		t.Error("authorization state is missing or inconsistent")
	}
}

func TestCurrentTrackMapsSpotifyPlayback(t *testing.T) {
	t.Parallel()

	client := testSpotifyClient(t)
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/me/player/currently-playing" {
			t.Errorf("Path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		return spotifyTestResponse(http.StatusOK, `{
			"is_playing": true,
			"progress_ms": 62000,
			"item": {
				"name": "Midnight City",
				"duration_ms": 243000,
				"artists": [{"name": "M83"}, {"name": "Guest"}],
				"album": {"images": [{"url": "https://image.example/cover.jpg"}]}
			}
		}`), nil
	})

	client.token = spotifyToken{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	track, err := client.CurrentTrack(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if track.Title != "Midnight City" || track.Artist != "M83, Guest" {
		t.Fatalf("unexpected track: %#v", track)
	}
	if !track.Playing || track.ElapsedSec != 62 || track.TotalSec != 243 {
		t.Fatalf("unexpected playback state: %#v", track)
	}
	if track.CoverURL != "https://image.example/cover.jpg" {
		t.Errorf("CoverURL = %q", track.CoverURL)
	}
}

func TestRefreshTokenUsesPKCEClientIDAndKeepsRefreshToken(t *testing.T) {
	t.Parallel()

	client := testSpotifyClient(t)
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/api/token":
			if request.Header.Get("Authorization") != "" {
				t.Errorf("Authorization = %q; want empty", request.Header.Get("Authorization"))
			}
			if err := request.ParseForm(); err != nil {
				return nil, err
			}
			if request.Form.Get("client_id") != "client-id" || request.Form.Get("refresh_token") != "old-refresh-token" {
				t.Errorf("unexpected refresh form: %v", request.Form)
			}
			return spotifyTestResponse(http.StatusOK, `{"access_token":"new-access-token","token_type":"Bearer","expires_in":3600}`), nil
		case "/me/player/currently-playing":
			return spotifyTestResponse(http.StatusNoContent, ""), nil
		default:
			t.Errorf("unexpected path: %s", request.URL.Path)
			return spotifyTestResponse(http.StatusNotFound, ""), nil
		}
	})

	client.token = spotifyToken{RefreshToken: "old-refresh-token"}
	if _, err := client.CurrentTrack(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.token.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q", client.token.AccessToken)
	}
	if client.token.RefreshToken != "old-refresh-token" {
		t.Errorf("RefreshToken = %q", client.token.RefreshToken)
	}
}

func TestDisconnectRemovesStoredSpotifyAccount(t *testing.T) {
	t.Parallel()

	client := testSpotifyClient(t)
	client.token = spotifyToken{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
	if err := client.saveToken(); err != nil {
		t.Fatal(err)
	}
	if !client.Connected() {
		t.Fatal("client should be connected before disconnecting")
	}
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if client.Connected() {
		t.Fatal("client remains connected")
	}
	if _, err := os.Stat(client.config.tokenFile); !os.IsNotExist(err) {
		t.Fatalf("token file still exists: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func spotifyTestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func testSpotifyClient(t *testing.T) *SpotifyClient {
	t.Helper()
	return &SpotifyClient{
		config: spotifyConfig{
			clientID:    "client-id",
			redirectURI: "http://127.0.0.1:8888/callback",
			tokenFile:   filepath.Join(t.TempDir(), "spotify-token.json"),
		},
		http:     &http.Client{Timeout: time.Second},
		accounts: "https://accounts.spotify.test",
		api:      "https://api.spotify.test",
	}
}
