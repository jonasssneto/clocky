package imaging

import (
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderCoverPlaceholderHasRequestedCellDimensions(t *testing.T) {
	rendered := RenderCover(nil, 8, 4)
	lines := strings.Split(rendered, "\n")
	if len(lines) != 4 {
		t.Fatalf("height = %d, want 4", len(lines))
	}
	for index, line := range lines {
		if ansi.StringWidth(line) != 8 {
			t.Errorf("line %d width = %d, want 8", index, ansi.StringWidth(line))
		}
	}
	if !strings.Contains(rendered, "♪") {
		t.Fatal("placeholder does not contain music symbol")
	}
}

func TestRenderCoverConvertsPixelsToTrueColorHalfBlocks(t *testing.T) {
	cover := image.NewRGBA(image.Rect(0, 0, 2, 4))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			cover.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	for y := 2; y < 4; y++ {
		for x := 0; x < 2; x++ {
			cover.Set(x, y, color.RGBA{B: 255, A: 255})
		}
	}
	rendered := RenderCover(cover, 3, 3)
	if len(strings.Split(rendered, "\n")) != 3 || !strings.Contains(rendered, "\x1b[38;2;") || !strings.Contains(rendered, "▀") {
		t.Fatalf("unexpected ANSI cover: %q", rendered)
	}
}

func TestDownloadAndRenderCoverUsesBoundedHTTPDecoder(t *testing.T) {
	imageData := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var encoded strings.Builder
	if err := jpeg.Encode(&encoded, imageData, nil); err != nil {
		t.Fatal(err)
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = coverRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		status, body := http.StatusOK, encoded.String()
		if request.URL.Path == "/missing" {
			status, body = http.StatusNotFound, "missing"
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	rendered, err := DownloadAndRenderCover("https://cover.test/image.jpg", 4, 3)
	if err != nil || len(strings.Split(rendered, "\n")) != 3 {
		t.Fatalf("DownloadAndRenderCover = %q, %v", rendered, err)
	}

	if _, err := DownloadAndRenderCover("https://cover.test/missing", 4, 3); err == nil || !strings.Contains(err.Error(), "HTTP status 404") {
		t.Fatalf("HTTP error = %v", err)
	}
	if _, err := DownloadAndRenderCover(":bad-url", 4, 3); err == nil {
		t.Fatal("invalid URL unexpectedly succeeded")
	}
}

type coverRoundTripFunc func(*http.Request) (*http.Response, error)

func (function coverRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
