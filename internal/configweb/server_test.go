package configweb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"clocky/internal/oauth"
)

func TestDashboardShowsIntegrationStatus(t *testing.T) {
	t.Parallel()

	integration := &fakeIntegration{connected: true}
	server := testServer(t, integration)
	request := httptest.NewRequest(http.MethodGet, server.PageURL(), nil)
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{"Clocky Configuration", "Spotify", "Connected", "Switch account", "Disconnect"} {
		if !strings.Contains(body, expected) {
			t.Errorf("dashboard does not contain %q", expected)
		}
	}
}

func TestConnectRedirectAndCallback(t *testing.T) {
	t.Parallel()

	integration := &fakeIntegration{}
	server := testServer(t, integration)
	connectForm := url.Values{"csrf_token": {server.csrfToken}}
	connectRequest := httptest.NewRequest(http.MethodPost, "/integrations/spotify/connect", strings.NewReader(connectForm.Encode()))
	connectRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	connectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(connectRecorder, connectRequest)

	if connectRecorder.Code != http.StatusSeeOther {
		t.Fatalf("connect status = %d", connectRecorder.Code)
	}
	if location := connectRecorder.Header().Get("Location"); location != "https://accounts.example/authorize" {
		t.Fatalf("connect redirect = %q", location)
	}

	callbackRequest := httptest.NewRequest(http.MethodGet, "/callback?code=authorization-code&state=expected-state", nil)
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
	invalidRequest := httptest.NewRequest(http.MethodPost, "/integrations/spotify/disconnect", nil)
	invalidRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusSeeOther {
		t.Fatalf("invalid CSRF status = %d", invalidRecorder.Code)
	}
	if !integration.Connected() {
		t.Fatal("invalid request disconnected the integration")
	}

	form := url.Values{"csrf_token": {server.csrfToken}}
	request := httptest.NewRequest(http.MethodPost, "/integrations/spotify/disconnect", strings.NewReader(form.Encode()))
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
