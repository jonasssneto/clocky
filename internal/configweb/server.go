// Package configweb serves Clocky's remote configuration dashboard.
package configweb

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"clocky/internal/oauth"
	"clocky/internal/settings"
	"clocky/internal/updater"
)

const (
	csrfTokenSize                  = 32
	scriptNonceSize                = 16
	marketSearchURL                = "https://brapi.dev/api/v2/tickers"
	marketSearchTimeout            = 8 * time.Second
	marketSearchBodyLimit          = 1 << 20
	minimumMarketQueryLength       = 2
	maximumMarketQueryLength       = 40
	marketSearchResultLimit        = "8"
	releaseListTimeout             = 12 * time.Second
	serverReadHeaderTimeout        = 5 * time.Second
	serverWriteTimeout             = 20 * time.Second
	serverIdleTimeout              = 30 * time.Second
	serverShutdownTimeout          = 2 * time.Second
	authorizationTimeout           = 15 * time.Second
	authorizationLifetime          = 10 * time.Minute
	maximumRSSFeeds                = 12
	maximumMarketSymbols           = 20
	maximumColorLength             = 16
	minimumVisualScale             = 50
	maximumVisualScale             = 100
	minimumBoxPadding              = 0
	maximumBoxPadding              = 2
	minimumColumnGap               = 0
	maximumColumnGap               = 4
	minimumChartHeight             = 1
	maximumChartHeight             = 4
	minimumNewsLimit               = 1
	maximumNewsLimit               = 20
	minimumRefreshMinutes          = 1
	maximumRefreshMinutes          = 120
	minimumRotationSeconds         = 10
	maximumRotationSeconds         = 3600
	minimumMarketRotationSeconds   = 1
	maximumMarketRotationSeconds   = 300
	minimumMarketFrameMilliseconds = 10
	maximumMarketFrameMilliseconds = 1000
	minimumMarqueeMilliseconds     = 80
	maximumMarqueeMilliseconds     = 1000
	minimumSpotifyPollSeconds      = 2
	maximumSpotifyPollSeconds      = 300
)

//go:embed dashboard.html result.html
var templateFiles embed.FS

type Server struct {
	listenAddress   string
	pageURL         string
	csrfToken       string
	templates       *template.Template
	integrations    map[string]oauth.Integration
	settings        *settings.Store
	marketSearchURL string
	httpClient      *http.Client
	mu              sync.RWMutex
	pending         map[string]pendingAuthorization
	updateRequests  chan UpdateRequest
}

type UpdateRequest struct {
	Version string
}

type pendingAuthorization struct {
	integration  oauth.Integration
	callbackPath string
	expiresAt    time.Time
}

type pageData struct {
	Notice       string
	Error        string
	CSRFToken    string
	ScriptNonce  string
	Integrations []integrationView
	Settings     settings.Snapshot
}

type integrationView struct {
	ID        string
	Name      string
	Initial   string
	Connected bool
}

type resultData struct {
	Success bool
	Title   string
	Message string
}

func New(pageURL string, integrations ...oauth.Integration) (*Server, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("parse configuration page URL: %w", err)
	}
	if parsed.Scheme != "http" || parsed.Host == "" || parsed.Port() == "" {
		return nil, errors.New("configuration page URL must be an absolute HTTP URL with a port")
	}
	if parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return nil, errors.New("configuration server only listens on a loopback address")
	}
	if parsed.User != nil {
		return nil, errors.New("configuration page URL must not include credentials")
	}
	parsed.Path = "/"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""

	csrfToken, err := randomToken(csrfTokenSize)
	if err != nil {
		return nil, fmt.Errorf("generate configuration CSRF token: %w", err)
	}
	templates, err := template.ParseFS(templateFiles, "dashboard.html", "result.html")
	if err != nil {
		return nil, fmt.Errorf("parse configuration templates: %w", err)
	}
	registered := make(map[string]oauth.Integration, len(integrations))
	visualSettings := settings.New()
	for _, integration := range integrations {
		if integration == nil || integration.ID() == "" || integration.Name() == "" {
			return nil, errors.New("configuration integrations require an ID and name")
		}
		if strings.ContainsAny(integration.ID(), "/?#") {
			return nil, fmt.Errorf("invalid configuration integration ID %q", integration.ID())
		}
		if _, exists := registered[integration.ID()]; exists {
			return nil, fmt.Errorf("duplicate configuration integration ID %q", integration.ID())
		}
		registered[integration.ID()] = integration
	}
	return &Server{
		listenAddress:   parsed.Host,
		pageURL:         parsed.String(),
		csrfToken:       csrfToken,
		templates:       templates,
		integrations:    registered,
		settings:        visualSettings,
		marketSearchURL: marketSearchURL,
		httpClient:      &http.Client{Timeout: marketSearchTimeout},
		pending:         make(map[string]pendingAuthorization),
		updateRequests:  make(chan UpdateRequest, 1),
	}, nil
}

func (s *Server) UpdateRequests() <-chan UpdateRequest {
	return s.updateRequests
}

func (s *Server) SetSettings(store *settings.Store) {
	if store != nil {
		s.settings = store
	}
}

func (s *Server) SetIntegration(integration oauth.Integration) {
	if integration == nil || integration.ID() == "" {
		return
	}
	s.mu.Lock()
	s.integrations[integration.ID()] = integration
	s.mu.Unlock()
}

func (s *Server) PageURL() string {
	return s.pageURL
}

func (s *Server) Serve(ctx context.Context) error {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("listen for configuration requests on %s: %w", s.listenAddress, err)
	}
	server := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- server.Serve(listener)
	}()

	select {
	case serveErr := <-serveDone:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("serve configuration dashboard: %w", serveErr)
		}
		return nil
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shut down configuration dashboard: %w", err)
		}
		return nil
	}
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		setSecurityHeaders(writer)
		switch {
		case request.URL.Path == "/":
			s.handleDashboard(writer, request)
		case request.URL.Path == "/settings":
			s.handleSettings(writer, request)
		case request.URL.Path == "/api/releases":
			s.handleReleases(writer, request)
		case request.URL.Path == "/api/update":
			s.handleUpdate(writer, request)
		case request.URL.Path == "/api/market-symbols":
			s.handleMarketSymbols(writer, request)
		case strings.HasPrefix(request.URL.Path, "/integrations/"):
			s.handleIntegration(writer, request)
		default:
			s.handleCallback(writer, request)
		}
	})
}

func (s *Server) handleUpdate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(writer, http.MethodPost)
		return
	}
	if err := request.ParseForm(); err != nil || request.Form.Get("csrf_token") != s.csrfToken {
		http.Error(writer, "Invalid configuration request", http.StatusForbidden)
		return
	}
	version := strings.TrimSpace(request.Form.Get("version"))
	if version == "" || len(version) > 64 || strings.ContainsAny(version, "\r\n") {
		http.Error(writer, "Select a valid release", http.StatusBadRequest)
		return
	}
	select {
	case s.updateRequests <- UpdateRequest{Version: version}:
		current := s.settings.Get()
		s.settings.SetUpdates(current.UpdatesEnabled, version)
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(writer).Encode(map[string]string{"status": "accepted"})
	default:
		http.Error(writer, "An update is already in progress", http.StatusConflict)
	}
}

type marketSymbolResult struct {
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	SubType string `json:"subType"`
}

func (s *Server) handleMarketSymbols(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	query := strings.TrimSpace(request.URL.Query().Get("q"))
	if len(query) < minimumMarketQueryLength || len(query) > maximumMarketQueryLength {
		http.Error(writer, "Enter between 2 and 40 characters", http.StatusBadRequest)
		return
	}
	values := url.Values{
		"search":    {query},
		"limit":     {marketSearchResultLimit},
		"sortBy":    {"volume"},
		"sortOrder": {"desc"},
	}
	switch request.URL.Query().Get("kind") {
	case "fii":
		values.Set("subType", "fii")
	case "stock":
		values.Set("subType", "stock")
	default:
		http.Error(writer, "Invalid market asset kind", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), marketSearchTimeout)
	defer cancel()
	providerRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, s.marketSearchURL+"?"+values.Encode(), nil)
	if err != nil {
		http.Error(writer, "Unable to prepare market search", http.StatusInternalServerError)
		return
	}
	providerRequest.Header.Set("Accept", "application/json")
	providerRequest.Header.Set("User-Agent", "Clocky configuration")
	response, err := s.httpClient.Do(providerRequest)
	if err != nil {
		http.Error(writer, "Market search is temporarily unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		http.Error(writer, "Market search is temporarily unavailable", http.StatusBadGateway)
		return
	}
	var payload struct {
		Results []marketSymbolResult `json:"results"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, marketSearchBodyLimit)).Decode(&payload); err != nil {
		http.Error(writer, "Market search returned an invalid response", http.StatusBadGateway)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(payload.Results)
}

func (s *Server) handleReleases(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), releaseListTimeout)
	defer cancel()
	releases, err := updater.ListReleases(ctx)
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		writer.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(writer).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(writer).Encode(releases)
}

func (s *Server) handleDashboard(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	integrations := make([]integrationView, 0, len(s.integrations))
	for _, integration := range s.integrations {
		initial := "?"
		for _, character := range integration.Name() {
			initial = strings.ToUpper(string(character))
			break
		}
		integrations = append(integrations, integrationView{
			ID:        integration.ID(),
			Name:      integration.Name(),
			Initial:   initial,
			Connected: integration.Connected(),
		})
	}
	sort.Slice(integrations, func(left, right int) bool { return integrations[left].Name < integrations[right].Name })
	scriptNonce, err := randomToken(scriptNonceSize)
	if err != nil {
		http.Error(writer, "Unable to prepare dashboard", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+scriptNonce+"'; style-src 'unsafe-inline'; connect-src 'self' https://nominatim.openstreetmap.org http://127.0.0.1:8888 http://localhost:8888; base-uri 'none'; frame-ancestors 'none'")
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates.ExecuteTemplate(writer, "dashboard.html", pageData{
		Notice:       request.URL.Query().Get("notice"),
		Error:        request.URL.Query().Get("error"),
		CSRFToken:    s.csrfToken,
		ScriptNonce:  scriptNonce,
		Integrations: integrations,
		Settings:     s.settings.Get(),
	})
}

func (s *Server) handleSettings(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(writer, http.MethodPost)
		return
	}
	if err := request.ParseForm(); err != nil || request.Form.Get("csrf_token") != s.csrfToken {
		s.redirectWithMessage(writer, request, "error", "This settings page expired. Reload it and try again.")
		return
	}
	current := s.settings.Get()
	visual := current.Visual
	visual.Scale = boundedFormInt(request, "scale", visual.Scale, minimumVisualScale, maximumVisualScale)
	visual.BoxPadding = boundedFormInt(request, "box_padding", visual.BoxPadding, minimumBoxPadding, maximumBoxPadding)
	visual.ColumnGap = boundedFormInt(request, "column_gap", visual.ColumnGap, minimumColumnGap, maximumColumnGap)
	visual.ChartHeight = boundedFormInt(request, "chart_height", visual.ChartHeight, minimumChartHeight, maximumChartHeight)
	visual.BorderColor = formColor(request, "border_color", visual.BorderColor)
	visual.TextColor = formColor(request, "text_color", visual.TextColor)
	visual.DimColor = formColor(request, "dim_color", visual.DimColor)
	visual.ValueColor = formColor(request, "value_color", visual.ValueColor)
	visual.AccentColor = formColor(request, "accent_color", visual.AccentColor)
	visual.Positive = formColor(request, "positive_color", visual.Positive)
	visual.Negative = formColor(request, "negative_color", visual.Negative)
	s.settings.SetVisual(visual)
	city := strings.TrimSpace(request.Form.Get("weather_city"))
	if city == "" {
		city = current.WeatherCity
	}
	country := strings.ToUpper(strings.TrimSpace(request.Form.Get("weather_country")))
	if len(country) != 2 {
		country = current.WeatherCountry
	}
	s.settings.SetWeather(city, country, strings.TrimSpace(request.Form.Get("weather_key")))
	s.settings.SetGitHub(strings.TrimSpace(request.Form.Get("github_user")))
	redirectURI := strings.TrimSpace(request.Form.Get("spotify_redirect_uri"))
	if redirectURI == "" {
		redirectURI = current.SpotifyRedirectURI
	}
	s.settings.SetSpotify(strings.TrimSpace(request.Form.Get("spotify_client_id")), redirectURI)
	feeds := make([]string, 0, maximumRSSFeeds)
	for _, value := range request.Form["rss_feeds"] {
		for _, line := range strings.Split(value, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && (strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "http://")) && len(feeds) < maximumRSSFeeds {
				feeds = append(feeds, line)
			}
		}
	}
	limit := boundedFormInt(request, "news_limit", current.NewsLimit, minimumNewsLimit, maximumNewsLimit)
	s.settings.SetRSS(feeds, limit)
	fiiSymbols := marketSymbols(strings.Join(request.Form["fii_symbols"], ","))
	stockSymbols := marketSymbols(strings.Join(request.Form["stock_symbols"], ","))
	s.settings.SetMarketSymbols(fiiSymbols, stockSymbols)
	s.settings.SetIntervals(
		boundedFormInt(request, "refresh_minutes", current.RefreshMinutes, minimumRefreshMinutes, maximumRefreshMinutes),
		boundedFormInt(request, "news_rotation_seconds", current.NewsRotationSeconds, minimumRotationSeconds, maximumRotationSeconds),
		boundedFormInt(request, "weather_rotation_seconds", current.WeatherRotationSeconds, minimumRotationSeconds, maximumRotationSeconds),
		boundedFormInt(request, "market_rotation_seconds", current.MarketRotationSeconds, minimumMarketRotationSeconds, maximumMarketRotationSeconds),
		boundedFormInt(request, "market_frame_ms", current.MarketFrameMilliseconds, minimumMarketFrameMilliseconds, maximumMarketFrameMilliseconds),
		boundedFormInt(request, "marquee_ms", current.MarqueeMilliseconds, minimumMarqueeMilliseconds, maximumMarqueeMilliseconds),
		boundedFormInt(request, "spotify_poll_seconds", current.SpotifyPollSeconds, minimumSpotifyPollSeconds, maximumSpotifyPollSeconds),
	)
	version := strings.TrimSpace(request.Form.Get("update_version"))
	if version == "" {
		version = "latest"
	}
	s.settings.SetUpdates(request.Form.Get("updates_enabled") == "on", version)
	s.redirectWithMessage(writer, request, "notice", "Dashboard settings saved.")
}

func marketSymbols(raw string) []string {
	symbols := make([]string, 0, maximumMarketSymbols)
	for _, token := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == ';' || r == ' ' || r == '\t' }) {
		symbol := strings.ToUpper(strings.TrimSpace(token))
		if symbol != "" && len(symbols) < maximumMarketSymbols {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

func boundedFormInt(request *http.Request, name string, fallback, minimum, maximum int) int {
	value, err := strconv.Atoi(request.Form.Get(name))
	if err != nil || value < minimum || value > maximum {
		return fallback
	}
	return value
}

func formColor(request *http.Request, name, fallback string) string {
	value := strings.TrimSpace(request.Form.Get(name))
	if value == "" || len(value) > maximumColorLength || strings.ContainsAny(value, "{};\n\r") {
		return fallback
	}
	return value
}

func (s *Server) handleIntegration(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		methodNotAllowed(writer, http.MethodPost)
		return
	}
	if err := request.ParseForm(); err != nil || request.Form.Get("csrf_token") != s.csrfToken {
		s.redirectWithMessage(writer, request, "error", "This settings page expired. Reload it and try again.")
		return
	}
	parts := strings.Split(strings.TrimPrefix(request.URL.Path, "/integrations/"), "/")
	if len(parts) != 2 {
		http.NotFound(writer, request)
		return
	}
	integration, exists := s.integrations[parts[0]]
	if !exists {
		http.NotFound(writer, request)
		return
	}

	switch parts[1] {
	case "connect":
		authorization, err := integration.BeginAuthorization()
		if err != nil {
			s.redirectWithMessage(writer, request, "error", err.Error())
			return
		}
		providerURL, parseErr := url.Parse(authorization.URL)
		if authorization.State == "" || authorization.CallbackPath == "" || parseErr != nil || providerURL.Scheme != "https" || providerURL.Host == "" {
			s.redirectWithMessage(writer, request, "error", "The integration returned an incomplete authorization request.")
			return
		}
		if authorization.CallbackPath == "/" || !strings.HasPrefix(authorization.CallbackPath, "/") || strings.HasPrefix(authorization.CallbackPath, "/integrations/") {
			s.redirectWithMessage(writer, request, "error", "The integration returned an invalid callback path.")
			return
		}
		s.mu.Lock()
		s.prunePendingLocked(time.Now(), integration.ID())
		s.pending[authorization.State] = pendingAuthorization{
			integration:  integration,
			callbackPath: authorization.CallbackPath,
			expiresAt:    time.Now().Add(authorizationLifetime),
		}
		s.mu.Unlock()
		http.Redirect(writer, request, authorization.URL, http.StatusSeeOther)
	case "disconnect":
		if err := integration.Disconnect(); err != nil {
			s.redirectWithMessage(writer, request, "error", err.Error())
			return
		}
		s.mu.Lock()
		s.prunePendingLocked(time.Now(), integration.ID())
		s.mu.Unlock()
		s.redirectWithMessage(writer, request, "notice", integration.Name()+" disconnected.")
	default:
		http.NotFound(writer, request)
	}
}

func (s *Server) handleCallback(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		methodNotAllowed(writer, http.MethodGet)
		return
	}
	state := request.URL.Query().Get("state")
	s.mu.Lock()
	s.prunePendingLocked(time.Now(), "")
	pending, exists := s.pending[state]
	if exists && pending.callbackPath == request.URL.Path {
		delete(s.pending, state)
	} else {
		exists = false
	}
	s.mu.Unlock()
	if !exists {
		http.NotFound(writer, request)
		return
	}
	if providerError := request.URL.Query().Get("error"); providerError != "" {
		s.renderResult(writer, http.StatusBadRequest, false, "Authorization declined", providerError)
		return
	}
	code := request.URL.Query().Get("code")
	if code == "" {
		s.renderResult(writer, http.StatusBadRequest, false, "Authorization failed", "The provider did not return an authorization code.")
		return
	}
	ctx, cancel := context.WithTimeout(request.Context(), authorizationTimeout)
	defer cancel()
	if err := pending.integration.CompleteAuthorization(ctx, code); err != nil {
		s.renderResult(writer, http.StatusBadGateway, false, "Connection failed", err.Error())
		return
	}
	s.renderResult(writer, http.StatusOK, true, pending.integration.Name()+" connected", "The integration is ready. You can return to the configuration dashboard.")
}

func (s *Server) prunePendingLocked(now time.Time, integrationID string) {
	for state, pending := range s.pending {
		if now.After(pending.expiresAt) || (integrationID != "" && pending.integration.ID() == integrationID) {
			delete(s.pending, state)
		}
	}
}

func (s *Server) renderResult(writer http.ResponseWriter, status int, success bool, title, message string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(status)
	_ = s.templates.ExecuteTemplate(writer, "result.html", resultData{Success: success, Title: title, Message: message})
}

func (s *Server) redirectWithMessage(writer http.ResponseWriter, request *http.Request, key, message string) {
	query := url.Values{key: {message}}
	http.Redirect(writer, request, "/?"+query.Encode(), http.StatusSeeOther)
}

func methodNotAllowed(writer http.ResponseWriter, allowed string) {
	writer.Header().Set("Allow", allowed)
	http.Error(writer, "Method not allowed", http.StatusMethodNotAllowed)
}

func setSecurityHeaders(writer http.ResponseWriter) {
	writer.Header().Set("Cache-Control", "no-store")
	// The dashboard is local-only, but users may open it as either
	// 127.0.0.1 or localhost. Both names resolve to the same local server.
	writer.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; connect-src 'self' http://127.0.0.1:8888 http://localhost:8888; base-uri 'none'; frame-ancestors 'none'")
	writer.Header().Set("Referrer-Policy", "no-referrer")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("X-Frame-Options", "DENY")
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
