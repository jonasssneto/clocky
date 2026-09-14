package configweb

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"clocky/internal/oauth"
	"clocky/internal/settings"
)

func TestDashboardShowsIntegrationStatus(t *testing.T) {
	t.Parallel()

	integration := &fakeIntegration{connected: true}
	server := testServer(t, integration)
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, server.PageURL(), nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"Clocky Configuration", "Spotify", "Connected", "Switch account", "Disconnect", "Dashboard appearance", "Current terminal"} {
		if !strings.Contains(body, expected) {
			t.Errorf("dashboard does not contain %q", expected)
		}
	}
}

func TestVisualSettingsAreSaved(t *testing.T) {
	t.Parallel()

	server := testServer(t, &fakeIntegration{})
	form := url.Values{
		"csrf_token":            {server.csrfToken},
		"scale":                 {"80"},
		"box_padding":           {"0"},
		"column_gap":            {"3"},
		"chart_height":          {"4"},
		"border_color":          {"#ffffff"},
		"accent_color":          {"#ff00aa"},
		"positive_color":        {"#00ff00"},
		"weather_city":          {"Curitiba"},
		"weather_key":           {"optional-key"},
		"github_user":           {"octocat"},
		"rss_feeds":             {"https://example.com/feed.xml\nhttps://example.org/rss"},
		"fii_symbols":           {"MXRF11", "VINO11", "KNCR11", "XPML11"},
		"stock_symbols":         {"VALE3", "PETR4"},
		"news_limit":            {"5"},
		"news_rotation_seconds": {"90"},
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("settings status = %d", recorder.Code)
	}
	visual := server.settings.Get().Visual
	if visual.Scale != 80 || visual.BoxPadding != 0 || visual.ColumnGap != 3 || visual.ChartHeight != 4 || visual.AccentColor != "#ff00aa" || server.settings.Get().WeatherCity != "Curitiba" || server.settings.Get().WeatherKey != "optional-key" || server.settings.Get().GitHubUser != "octocat" || server.settings.Get().NewsLimit != 5 || server.settings.Get().NewsRotationSeconds != 90 {
		t.Fatalf("unexpected visual settings: %+v", visual)
	}
	if got := strings.Join(server.settings.Get().FiiSymbols, ","); got != "MXRF11,VINO11,KNCR11,XPML11" {
		t.Fatalf("FII symbols = %q", got)
	}
	if got := strings.Join(server.settings.Get().StockSymbols, ","); got != "VALE3,PETR4" {
		t.Fatalf("stock symbols = %q", got)
	}
}

func TestMarketSymbolSearchFiltersFIIs(t *testing.T) {
	t.Parallel()

	server := testServer(t, &fakeIntegration{})
	server.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Query().Get("search") != "mxrf" || request.URL.Query().Get("subType") != "fii" || request.URL.Query().Get("limit") != "8" {
			return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("unexpected query"))}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"results":[{"symbol":"MXRF11","name":"Maxi Renda","subType":"fii"}]}`)),
		}, nil
	})}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/market-symbols?q=mxrf&kind=fii", nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{"MXRF11", "Maxi Renda", `"subType":"fii"`} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("market search does not contain %q", expected)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestConnectRedirectAndCallback(t *testing.T) {
	t.Parallel()

	integration := &fakeIntegration{}
	server := testServer(t, integration)
	connectForm := url.Values{"csrf_token": {server.csrfToken}}
	connectRequest := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/integrations/spotify/connect", strings.NewReader(connectForm.Encode()))
	connectRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	connectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(connectRecorder, connectRequest)

	if connectRecorder.Code != http.StatusSeeOther {
		t.Fatalf("connect status = %d", connectRecorder.Code)
	}
	if location := connectRecorder.Header().Get("Location"); location != "https://accounts.example/authorize" {
		t.Fatalf("connect redirect = %q", location)
	}

	callbackRequest := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/callback?code=authorization-code&state=expected-state", nil)
	callbackRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(callbackRecorder, callbackRequest)
	if callbackRecorder.Code != http.StatusOK {
		t.Fatalf("callback status = %d; body = %s", callbackRecorder.Code, callbackRecorder.Body.String())
	}
	if integration.authorizationCode() != "authorization-code" || !integration.Connected() {
		t.Fatal("callback did not complete the integration")
	}
}

func TestDisconnectRequiresCSRFAndClearsIntegration(t *testing.T) {
	t.Parallel()

	integration := &fakeIntegration{connected: true}
	server := testServer(t, integration)
	invalidRequest := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/integrations/spotify/disconnect", nil)
	invalidRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusSeeOther {
		t.Fatalf("invalid CSRF status = %d", invalidRecorder.Code)
	}
	if !integration.Connected() {
		t.Fatal("invalid request disconnected the integration")
	}

	form := url.Values{"csrf_token": {server.csrfToken}}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/integrations/spotify/disconnect", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther {
		t.Fatalf("disconnect status = %d", recorder.Code)
	}
	if integration.Connected() {
		t.Fatal("integration remains connected")
	}
}

func testServer(t *testing.T, integration oauth.Integration) *Server {
	t.Helper()
	server, err := New("http://127.0.0.1:8888/", integration)
	if err != nil {
		t.Fatal(err)
	}
	// Keep parallel handler tests from persisting settings to the user's real config.
	server.settings = &settings.Store{Snapshot: server.settings.Get()}
	return server
}

type fakeIntegration struct {
	mu        sync.Mutex
	connected bool
	code      string
}

func (integration *fakeIntegration) ID() string   { return "spotify" }
func (integration *fakeIntegration) Name() string { return "Spotify" }

func (integration *fakeIntegration) Connected() bool {
	integration.mu.Lock()
	defer integration.mu.Unlock()
	return integration.connected
}

func (integration *fakeIntegration) BeginAuthorization() (oauth.Authorization, error) {
	return oauth.Authorization{
		URL:          "https://accounts.example/authorize",
		State:        "expected-state",
		CallbackPath: "/callback",
	}, nil
}

func (integration *fakeIntegration) CompleteAuthorization(_ context.Context, code string) error {
	integration.mu.Lock()
	defer integration.mu.Unlock()
	integration.code = code
	integration.connected = true
	return nil
}

func (integration *fakeIntegration) Disconnect() error {
	integration.mu.Lock()
	defer integration.mu.Unlock()
	integration.connected = false
	return nil
}

func (integration *fakeIntegration) authorizationCode() string {
	integration.mu.Lock()
	defer integration.mu.Unlock()
	return integration.code
}
