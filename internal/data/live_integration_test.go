package data

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLiveDataAdaptersMapProviderResponses(t *testing.T) { //nolint:gocyclo // This integration test validates several related provider mappings.
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		status := http.StatusOK
		body := ""
		switch request.URL.Host {
		case "geocoding-api.open-meteo.com":
			body = `{"results":[{"name":"Curitiba","latitude":-25.43,"longitude":-49.27,"timezone":"America/Sao_Paulo"}]}`
		case "api.open-meteo.com":
			body = `{
				"current":{"temperature_2m":21.6,"apparent_temperature":20.5,"relative_humidity_2m":71.4,"wind_speed_10m":10.6,"wind_direction_10m":90,"weather_code":2},
				"daily":{"weather_code":[2,61],"temperature_2m_max":[25.6,22.4],"temperature_2m_min":[15.5,14.4],"uv_index_max":[6.7],"sunrise":["2026-09-14T06:03"],"sunset":["2026-09-14T18:07"]},
				"hourly":{"time":["2026-09-14T10:00","2026-09-14T11:00"],"precipitation_probability":[20,80],"precipitation":[0,1]}
			}`
		case "air-quality-api.open-meteo.com":
			body = `{"current":{"us_aqi":42}}`
		case "date.nager.at":
			date := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
			body = fmt.Sprintf(`[{"date":%q,"localName":"Test Holiday","name":"Holiday"}]`, date)
		case "github-contributions-api.jogruber.de":
			body = githubPayload(35)
		case "capitalagora.com.br":
			if request.URL.Path == "/api/ativo/BAD11" {
				status = http.StatusNotFound
				break
			}
			body = `{"ticker":"MXRF11","preco":10.31,"faixa52":{"minima":9.5,"maxima":11.2},"variacoes":{"dia":0.49},"indicadores":{"dy_12m":12.1}}`
		default:
			return nil, fmt.Errorf("unexpected host %q", request.URL.Host)
		}
		return testHTTPResponse(status, body), nil
	})
	t.Cleanup(func() { http.DefaultTransport = originalTransport })

	today, tomorrow, overview, err := FetchLiveWeather(context.Background(), "Curitiba", "br", "unused")
	if err != nil {
		t.Fatal(err)
	}
	if today.Condition != "Partly cloudy" || today.TempC != 22 || today.FeelsLike != 21 || today.Wind != "← 11 km/h" || today.Humidity != 71 {
		t.Fatalf("today weather = %+v", today)
	}
	if tomorrow.Condition != "Rain" || tomorrow.TempHigh != 22 || tomorrow.TempLow != 14 {
		t.Fatalf("tomorrow weather = %+v", tomorrow)
	}
	if overview.UVIndex != 7 || overview.UVLevel != "high" || overview.AirQuality != "good" || overview.Holiday != "Test Holiday" {
		t.Fatalf("today overview = %+v", overview)
	}

	days, total, err := FetchGitHubContributions(context.Background(), " octocat ")
	if err != nil || len(days) != 30 || total != 1234 || days[0].Label != "06" || days[0].Level != 4 {
		t.Fatalf("GitHub contributions = %d days, total %d, err %v, first %+v", len(days), total, err, days[0])
	}
	if _, _, err := FetchGitHubContributions(context.Background(), " "); err == nil {
		t.Fatal("empty GitHub username unexpectedly succeeded")
	}

	assets, err := FetchMarketFunds(context.Background(), []string{" mxrf11 ", "MXRF11", "BAD11"})
	if err != nil || len(assets) != 1 {
		t.Fatalf("market assets = %#v, %v", assets, err)
	}
	asset := assets[0]
	if asset.Symbol != "MXRF11" || asset.Price != 10.31 || asset.DividendYield != "12.1%" || len(asset.History) != 3 {
		t.Fatalf("market asset = %+v", asset)
	}
	if assets, err := FetchMarketFunds(context.Background(), []string{"BAD11"}); err == nil || assets != nil {
		t.Fatalf("all-failed markets = %#v, %v", assets, err)
	}
}

func TestProviderHelpersHandleInvalidResponses(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/status":
			return testHTTPResponse(http.StatusBadGateway, "bad gateway"), nil
		case "/invalid":
			return testHTTPResponse(http.StatusOK, "not-json"), nil
		case "/empty-location":
			return testHTTPResponse(http.StatusOK, `{"results":[]}`), nil
		case "/empty-feed":
			return testHTTPResponse(http.StatusOK, `<rss><channel/></rss>`), nil
		default:
			return nil, errors.New("transport failure")
		}
	})}
	var target map[string]any
	if err := getJSON(context.Background(), client, "https://test.local/status", &target); err == nil {
		t.Fatal("non-success weather response unexpectedly succeeded")
	}
	if err := getJSON(context.Background(), client, "https://test.local/invalid", &target); err == nil {
		t.Fatal("invalid weather JSON unexpectedly succeeded")
	}
	if _, err := fetchLocation(context.Background(), client, "Nowhere"); err == nil {
		t.Fatal("empty geocoding response unexpectedly succeeded")
	}
	if _, err := fetchRSSFeed(context.Background(), client, "https://test.local/empty-feed"); err == nil {
		t.Fatal("empty RSS feed unexpectedly succeeded")
	}
	if _, err := fetchNextHoliday(context.Background(), client, "Brazil", time.Now()); err == nil {
		t.Fatal("invalid country unexpectedly succeeded")
	}
}

func githubPayload(count int) string {
	var contributions strings.Builder
	for index := 1; index <= count; index++ {
		if index > 1 {
			contributions.WriteByte(',')
		}
		fmt.Fprintf(&contributions, `{"date":%q,"count":%d,"level":%d}`, fmt.Sprintf("%02d", index), index, index%7)
	}
	return `{"total":{"lastYear":1234},"contributions":[` + contributions.String() + `]}`
}

func testHTTPResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
