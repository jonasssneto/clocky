package data

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMockDataReturnsIndependentRealisticValues(t *testing.T) {
	today, tomorrow, news, contributions := FetchAll()
	if today.Condition == "" || tomorrow.Condition == "" || len(news) < 3 || len(contributions) != 30 {
		t.Fatalf("incomplete mock data: today=%+v tomorrow=%+v news=%d contributions=%d", today, tomorrow, len(news), len(contributions))
	}
	if len(MockStocks()[0].History) < 3 || len(MockFunds()[0].History) < 3 {
		t.Fatal("market mocks need enough history for responsive charts")
	}

	news[0].Title = "changed"
	contributions[0].Count = 99
	stocks := MockStocks()
	stocks[0].History[0] = -1
	if MockNews()[0].Title == "changed" || MockContributions()[0].Count == 99 || MockStocks()[0].History[0] == -1 {
		t.Fatal("mock collections share mutable state")
	}
}

func TestWeatherHelpers(t *testing.T) {
	for _, test := range []struct {
		input float64
		want  int
	}{
		{input: 21.6, want: 22},
		{input: -2.6, want: -2},
	} {
		if got := roundedProviderValue(test.input); got != test.want {
			t.Errorf("roundedProviderValue(%v) = %d, want %d", test.input, got, test.want)
		}
	}
	if got := rainProbability([]int{10, 80, 20, 90}, 1); got != 90 {
		t.Fatalf("rainProbability = %d, want 90", got)
	}
	if got := rainProbability([]int{10}, 2); got != 0 {
		t.Fatalf("out-of-range rainProbability = %d", got)
	}
	times := []string{"2026-09-14T10:00", "2026-09-14T11:00", "short"}
	if got := rainTime(times, []int{10, 70, 90}, []float64{0, 1, 0}, 0); got != "11:00" {
		t.Fatalf("rainTime = %q, want 11:00", got)
	}
	if got := rainTime([]string{"brief"}, []int{50}, []float64{1}, 0); got != "brief" {
		t.Fatalf("short rainTime = %q", got)
	}
	if got := rainTime(nil, nil, nil, 0); got != "" {
		t.Fatalf("empty rainTime = %q", got)
	}

	for _, test := range []struct {
		input int
		want  string
	}{{0, "low"}, {3, "moderate"}, {6, "high"}, {8, "very high"}, {11, "extreme"}} {
		if got := uvLabel(test.input); got != test.want {
			t.Errorf("uvLabel(%d) = %q, want %q", test.input, got, test.want)
		}
	}
	for _, test := range []struct {
		input float64
		want  string
	}{{20, "good"}, {75, "moderate"}, {125, "unhealthy for sensitive groups"}, {175, "unhealthy"}} {
		if got := airQualityLabel(test.input); got != test.want {
			t.Errorf("airQualityLabel(%v) = %q, want %q", test.input, got, test.want)
		}
	}
	if windArrow(0) != "↓" || windArrow(90) != "←" || formatClock(nil) != "--:--" || formatClock([]string{"invalid"}) != "--:--" {
		t.Fatal("weather formatting helpers returned unexpected values")
	}
	conditions := []struct {
		code int
		want string
	}{{0, "Clear sky"}, {2, "Partly cloudy"}, {45, "Foggy"}, {61, "Rain"}, {71, "Snow"}, {95, "Thunderstorm"}}
	for _, test := range conditions {
		condition, icon := weatherDescription(test.code)
		if condition != test.want || len(icon) != 4 {
			t.Errorf("weatherDescription(%d) = %q, %d icon rows", test.code, condition, len(icon))
		}
	}
}

func TestMarketHelpers(t *testing.T) {
	if got := marketHistory(8, 10, 12); !reflect.DeepEqual(got, []float64{8, 10, 12}) {
		t.Fatalf("marketHistory = %v", got)
	}
	if got := marketHistory(0, 10, 10); !reflect.DeepEqual(got, []float64{10}) {
		t.Fatalf("marketHistory filters invalid and duplicate values: %v", got)
	}
	if formatYield(0) != "—" || formatYield(12.34) != "12.3%" {
		t.Fatal("unexpected yield formatting")
	}
	assets, err := FetchMarketFunds(context.Background(), []string{"", "too-long-market-symbol"})
	if err != nil || assets != nil {
		t.Fatalf("empty normalized symbols = %v, %v", assets, err)
	}
}

func TestFetchMarketAssetClosesResponseBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		wantOK bool
	}{
		{name: "valid quote", status: http.StatusOK, body: `{"ticker":"MXRF11","preco":10.31}`, wantOK: true},
		{name: "HTTP error", status: http.StatusBadGateway, body: "bad gateway"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackingReadCloser{Reader: strings.NewReader(test.body)}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: body}, nil
			})}
			_, ok := fetchMarketAsset(context.Background(), client, "MXRF11")
			if ok != test.wantOK {
				t.Fatalf("fetchMarketAsset success = %v, want %v", ok, test.wantOK)
			}
			if !body.closed {
				t.Fatal("market response body was not closed")
			}
		})
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestFetchRSSParsesSortsAndLimitsRSS(t *testing.T) {
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("User-Agent") != "Clocky RSS reader" {
			t.Errorf("User-Agent = %q", request.Header.Get("User-Agent"))
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<?xml version="1.0"?><rss><channel><title>Test News</title>
			<item><title>Older</title><description> First summary </description><pubDate>Sun, 13 Sep 2026 10:00:00 -0300</pubDate></item>
			<item><title>Newest</title><description>Second</description><pubDate>Mon, 14 Sep 2026 10:00:00 -0300</pubDate></item>
		</channel></rss>`))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	items, err := FetchRSS(context.Background(), []string{"https://feed.test/rss"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Newest" || items[0].Source != "Test News" || items[0].Summary != "Second" {
		t.Fatalf("unexpected RSS items: %#v", items)
	}
}

func TestFetchRSSCombinesFeedsAndHandlesFailures(t *testing.T) {
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status := http.StatusOK
		body := ""
		switch request.URL.Host {
		case "older.test":
			body = `<feed><title>Older Feed</title><entry><title>Older</title><summary>summary</summary><published>2026-09-13T10:00:00Z</published></entry></feed>`
		case "newer.test":
			body = `<feed><title>Newer Feed</title><entry><title>Newer</title><summary>summary</summary><published>2026-09-14T10:00:00Z</published></entry></feed>`
		default:
			status = http.StatusBadGateway
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	items, err := FetchRSS(context.Background(), []string{"https://older.test/rss", "https://broken.test/rss", "https://newer.test/rss"}, 10)
	if err != nil || len(items) != 2 || items[0].Title != "Newer" || items[1].Title != "Older" {
		t.Fatalf("combined feeds = %#v, %v", items, err)
	}
	if items, err := FetchRSS(context.Background(), []string{"https://broken.test/rss"}, 3); err == nil || items != nil {
		t.Fatalf("broken feed = %#v, %v", items, err)
	}
	if items, err := FetchRSS(context.Background(), nil, 3); err != nil || items != nil {
		t.Fatalf("empty feeds = %#v, %v", items, err)
	}
}

func TestRSSHelpers(t *testing.T) {
	if maxInt(2, 1) != 2 || maxInt(1, 2) != 2 || firstNonEmpty(" ", "value") != "value" {
		t.Fatal("basic RSS helper failed")
	}
	if !parseFeedDate("2026-09-14T10:00:00Z").Equal(time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("RFC3339 feed date was not parsed")
	}
	if !parseFeedDate("invalid").IsZero() {
		t.Fatal("invalid feed date should be zero")
	}
	if got := feedSource("https://news.example/feed", " "); got != "news.example" {
		t.Fatalf("feedSource = %q", got)
	}
	if got := feedSource(":bad", " "); got != "RSS" {
		t.Fatalf("invalid feed source = %q", got)
	}
	if !strings.Contains((&spotifyHTTPError{operation: "request", status: 400, message: "bad"}).Error(), "HTTP 400") {
		t.Fatal("Spotify HTTP error omits status")
	}
}
