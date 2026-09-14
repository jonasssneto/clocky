// Package configweb serves Clocky's remote configuration dashboard.
package configweb

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
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
)

//go:embed dashboard.html result.html
var templateFiles embed.FS

type Server struct {
	listenAddress string
	pageURL       string
	csrfToken     string
	templates     *template.Template
	integrations  map[string]oauth.Integration
	settings      *settings.Store
	mu            sync.RWMutex
	pending       map[string]pendingAuthorization
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

	csrfToken, err := randomToken(32)
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
		listenAddress: parsed.Host,
		pageURL:       parsed.String(),
		csrfToken:     csrfToken,
		templates:     templates,
		integrations:  registered,
		settings:      visualSettings,
		pending:       make(map[string]pendingAuthorization),
	}, nil
}

func (s *Server) SetSettings(store *settings.Store) {
	if store != nil {
		s.settings = store
	}
}

func (s *Server) PageURL() string {
	return s.pageURL
}

func (s *Server) Serve(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		return fmt.Errorf("listen for configuration requests on %s: %w", s.listenAddress, err)
	}
	server := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       30 * time.Second,
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
		shutdownContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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
		case strings.HasPrefix(request.URL.Path, "/integrations/"):
			s.handleIntegration(writer, request)
		default:
			s.handleCallback(writer, request)
		}
	})
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
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates.ExecuteTemplate(writer, "dashboard.html", pageData{
		Notice:       request.URL.Query().Get("notice"),
		Error:        request.URL.Query().Get("error"),
		CSRFToken:    s.csrfToken,
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
	current := s.settings.Get().Visual
	visual := current
	visual.Scale = boundedFormInt(request, "scale", current.Scale, 50, 100)
	visual.BoxPadding = boundedFormInt(request, "box_padding", current.BoxPadding, 0, 2)
	visual.ColumnGap = boundedFormInt(request, "column_gap", current.ColumnGap, 0, 4)
	visual.ChartHeight = boundedFormInt(request, "chart_height", current.ChartHeight, 1, 4)
	visual.BorderColor = formColor(request, "border_color", current.BorderColor)
	visual.TextColor = formColor(request, "text_color", current.TextColor)
	visual.DimColor = formColor(request, "dim_color", current.DimColor)
	visual.ValueColor = formColor(request, "value_color", current.ValueColor)
	visual.AccentColor = formColor(request, "accent_color", current.AccentColor)
	visual.Positive = formColor(request, "positive_color", current.Positive)
	visual.Negative = formColor(request, "negative_color", current.Negative)
	s.settings.SetVisual(visual)
	city := strings.TrimSpace(request.Form.Get("weather_city"))
	if city == "" {
		city = s.settings.Get().WeatherCity
	}
	country := strings.ToUpper(strings.TrimSpace(request.Form.Get("weather_country")))
	if len(country) != 2 {
		country = s.settings.Get().WeatherCountry
	}
	s.settings.SetWeather(city, country, strings.TrimSpace(request.Form.Get("weather_key")))
	s.settings.SetGitHub(strings.TrimSpace(request.Form.Get("github_user")))
	feeds := make([]string, 0, 12)
	for _, line := range strings.Split(request.Form.Get("rss_feeds"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "https://") || strings.HasPrefix(line, "http://")) && len(feeds) < 12 {
			feeds = append(feeds, line)
		}
	}
	limit := boundedFormInt(request, "news_limit", s.settings.Get().NewsLimit, 1, 20)
	s.settings.SetRSS(feeds, limit)
	fiiSymbols := marketSymbols(request.Form.Get("fii_symbols"))
	stockSymbols := marketSymbols(request.Form.Get("stock_symbols"))
	s.settings.SetMarketSymbols(fiiSymbols, stockSymbols)
	s.settings.SetIntervals(
		boundedFormInt(request, "refresh_minutes", s.settings.Get().RefreshMinutes, 1, 120),
		boundedFormInt(request, "news_rotation_seconds", s.settings.Get().NewsRotationSeconds, 10, 3600),
		boundedFormInt(request, "weather_rotation_seconds", s.settings.Get().WeatherRotationSeconds, 10, 3600),
		boundedFormInt(request, "market_rotation_seconds", s.settings.Get().MarketRotationSeconds, 1, 300),
		boundedFormInt(request, "market_frame_ms", s.settings.Get().MarketFrameMilliseconds, 10, 1000),
		boundedFormInt(request, "marquee_ms", s.settings.Get().MarqueeMilliseconds, 80, 1000),
		boundedFormInt(request, "spotify_poll_seconds", s.settings.Get().SpotifyPollSeconds, 2, 300),
	)
	s.redirectWithMessage(writer, request, "notice", "Dashboard settings saved.")
}

func marketSymbols(raw string) []string {
	symbols := make([]string, 0, 20)
	for _, token := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == ';' || r == ' ' || r == '\t' }) {
		symbol := strings.ToUpper(strings.TrimSpace(token))
		if symbol != "" && len(symbols) < 20 {
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
	if value == "" || len(value) > 16 || strings.ContainsAny(value, "{};\n\r") {
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
			expiresAt:    time.Now().Add(10 * time.Minute),
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
	ctx, cancel := context.WithTimeout(request.Context(), 15*time.Second)
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
