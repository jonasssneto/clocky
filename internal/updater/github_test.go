package updater

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

func TestListReleasesFiltersDraftsAndMapsAssets(t *testing.T) {
	originalTransport := http.DefaultTransport
	http.DefaultTransport = updaterRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Accept") != "application/vnd.github+json" || request.URL.Query().Get("per_page") != "30" {
			t.Errorf("unexpected release request: %s, headers %v", request.URL, request.Header)
		}
		body := `[
			{"tag_name":"v2","name":"Version 2","draft":true},
			{"tag_name":"v1","name":"Version 1","assets":[{"name":"clocky-linux-arm64","browser_download_url":"https://example.test/download"}]}
		]`
		return updaterResponse(http.StatusOK, body), nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	releases, err := ListReleases(context.Background())
	if err != nil || len(releases) != 1 || releases[0].TagName != "v1" {
		t.Fatalf("releases = %#v, %v", releases, err)
	}
}

func TestListReleasesHandlesNotFoundAndInvalidJSON(t *testing.T) {
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = updaterRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return updaterResponse(http.StatusNotFound, "missing"), nil
	})
	if releases, err := ListReleases(context.Background()); err != nil || len(releases) != 0 {
		t.Fatalf("not found = %#v, %v", releases, err)
	}
	http.DefaultTransport = updaterRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return updaterResponse(http.StatusOK, "invalid"), nil
	})
	if _, err := ListReleases(context.Background()); err == nil {
		t.Fatal("invalid release JSON unexpectedly succeeded")
	}
}

func TestAssetForCurrentPlatform(t *testing.T) {
	wanted := "clocky-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm" {
		wanted += "v7"
	}
	if runtime.GOOS == "windows" {
		wanted += ".exe"
	}
	release := Release{Assets: []Asset{{Name: "checksums.txt"}, {Name: "release-" + wanted, URL: "download"}}}
	asset, ok := AssetForCurrentPlatform(release)
	if !ok || asset.URL != "download" {
		t.Fatalf("asset = %+v, found %v", asset, ok)
	}
	if _, ok := AssetForCurrentPlatform(Release{}); ok {
		t.Fatal("missing platform asset was reported as found")
	}
}

type updaterRoundTripFunc func(*http.Request) (*http.Response, error)

func (function updaterRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func updaterResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
