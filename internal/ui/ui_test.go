package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"clocky/internal/data"
	"clocky/internal/settings"
)

func TestTextHelpersAreANSIWidthAware(t *testing.T) {
	marqueeState.Lock()
	marqueeState.offset = 0
	marqueeState.Unlock()
	if got := truncateLine("  hello world  ", 8); ansi.StringWidth(got) != 8 || got != "hello wo" {
		t.Fatalf("truncateLine = %q, width %d", got, ansi.StringWidth(got))
	}
	advanceMarquee()
	if got := fitMarquee("hello world", 5); got != "ello " || ansi.StringWidth(got) != 5 {
		t.Fatalf("advanced marquee = %q", got)
	}
	if got := truncateLine("hello", 0); got != "" {
		t.Fatalf("zero-width line = %q", got)
	}
	if got := fitLine("short", 8); lipgloss.Width(got) != 8 {
		t.Fatalf("fitLine width = %d", lipgloss.Width(got))
	}
	if got := fitStaticLine("a long static line", 7); lipgloss.Width(got) != 7 || !strings.Contains(got, "…") {
		t.Fatalf("fitStaticLine = %q, width %d", got, lipgloss.Width(got))
	}
}

func TestClockWeatherAndNewsResponsiveRendering(t *testing.T) {
	resetVisuals()
	now := time.Date(2026, 9, 14, 12, 34, 56, 0, time.UTC)
	if rows := bigText("12:x"); len(rows) != 5 || !strings.Contains(strings.Join(rows, ""), "███") {
		t.Fatalf("unexpected big clock glyphs: %q", rows)
	}
	large := renderClock(now, 80)
	compact := renderClock(now, 15)
	if !strings.Contains(large, "██") || strings.Contains(compact, "██") || !strings.Contains(compact, "12:34:56") {
		t.Fatal("clock did not switch between large and compact renderers")
	}

	today := data.MockToday()
	tomorrow := data.MockTomorrow()
	tomorrow.RainProbability = 70
	tomorrow.RainTime = "14:00"
	wide := renderWeather(today, tomorrow, 80, "Curitiba", false)
	narrow := renderWeather(today, tomorrow, 20, "Curitiba", false)
	rotated := renderWeather(today, tomorrow, 40, "Curitiba", true)
	if !strings.Contains(wide, "Feels like") || lipgloss.Width(narrow) > 20-boxStyle.GetHorizontalFrameSize() || !strings.Contains(rotated, "Tomorrow") || !strings.Contains(rotated, "70% at 14:00") {
		t.Fatalf("unexpected responsive weather: wide=%q narrow=%q tomorrow=%q", wide, narrow, rotated)
	}
	if weatherTitle(" ", "Today") != "Weather · São Paulo · Today" {
		t.Fatal("empty city did not use default weather title")
	}

	wrapped := wrapNewsTitle("one extraordinarilylongword three", 6)
	if len(wrapped) < 4 {
		t.Fatalf("long news word was not wrapped: %v", wrapped)
	}
	news := renderNewsBox(34, 8, now, data.MockNews(), 3)
	if !strings.Contains(news, "News") || !strings.Contains(news, "Government") || lipgloss.Width(news) > 34 || lipgloss.Height(news) > 8 {
		t.Fatalf("unexpected news card dimensions/content: %dx%d", lipgloss.Width(news), lipgloss.Height(news))
	}
	if got := renderNewsBox(20, boxStyle.GetVerticalFrameSize(), now, nil, 3); got != "" {
		t.Fatalf("too-short news card = %q", got)
	}
}

func TestTodayGithubMarketsAndSpotifyCards(t *testing.T) {
	resetVisuals()
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	overview := data.MockTodayOverview()
	overview.UpdatedAt = now
	overview.RainProbability = 80
	overview.RainTime = "15:00"
	if daylight := renderDaylight(40, now, overview); !strings.Contains(daylight, overview.Sunrise) || !strings.Contains(daylight, overview.Sunset) {
		t.Fatalf("daylight = %q", daylight)
	}
	if compact := renderTodayBox(22, boxStyle.GetVerticalFrameSize()+1, now, overview); !strings.Contains(compact, "Hol") {
		t.Fatalf("compact today card = %q", compact)
	}
	full := renderTodayBox(42, 10, now, overview)
	for _, expected := range []string{"Today", "Min 20°C", "Rain", "Holiday", "Air"} {
		if !strings.Contains(full, expected) {
			t.Errorf("today card missing %q", expected)
		}
	}

	days := data.MockContributions()
	monthly := renderGithubBox(60, 8, now, days, 1234)
	compactGithub := renderGithubBox(22, 6, now, days, 0)
	if !strings.Contains(monthly, "last 30 days") || !strings.Contains(compactGithub, "Git") || dayLabel("2026-09-14") != "14" || dayLabel("7") != "7" {
		t.Fatal("GitHub cards did not render monthly and compact variants")
	}

	stocks, funds := data.MockStocks(), data.MockFunds()
	chart := renderLineChart(stocks[0].History, 12, 2, stocks[0].Change)
	flatChart := renderLineChart([]float64{1}, 3, 1, 0)
	if len(chart) != 2 || len(flatChart) != 1 || renderLineChart(nil, 0, 1, 0) != nil {
		t.Fatal("line chart dimensions are incorrect")
	}
	if got := formatBrazilianReal(1234.5); got != "R$ 1234,50" {
		t.Fatalf("currency = %q", got)
	}
	panel := renderMarketsBox(52, 8, 0, 1, 0, false, stocks, funds)
	sliding := renderMarketsBox(52, 8, 0, 1, marketSlideSteps/2, true, stocks, funds)
	empty := renderMarketPanel(30, 3, 0, nil, nil)
	if !strings.Contains(panel, "Funds") || !strings.Contains(panel, "Stocks") || sliding == panel || !strings.Contains(empty[0], "not configured") {
		t.Fatal("market panel states were not rendered")
	}
	if got := renderMarketRange(data.MarketAsset{}, 20); !strings.Contains(got[0], "unavailable") {
		t.Fatalf("missing range = %q", got)
	}

	track := data.Track{Title: "A very long track title", Artist: "Artist", Playing: true, ElapsedSec: 62, TotalSec: 243}
	spotifyWide := renderSpotifyBox(40, 8, track, "", "", "")
	spotifyNarrow := renderSpotifyBox(16, 7, track, "", "", "")
	loggedOut := renderSpotifyBox(36, 8, data.Track{}, "", "Connect Spotify", "http://127.0.0.1:8888/")
	if !strings.Contains(spotifyWide, "1:02 / 4:03") || !strings.Contains(spotifyNarrow, "1:02/4:03") || !strings.Contains(loggedOut, "Connect Spotify") {
		t.Fatal("Spotify variants omitted playback or login information")
	}
	if got := renderProgressBar(10, -1, 100); lipgloss.Width(got) != 10 {
		t.Fatalf("progress width = %d", lipgloss.Width(got))
	}
	if got := renderPlaybackLine("Artist", 1, false); !strings.Contains(got, "▶") {
		t.Fatalf("tiny playback line = %q", got)
	}
}

func TestUpdateHandlesDataAnimationAndErrors(t *testing.T) { //nolint:gocyclo // The state-machine sequence is clearer as one integration test.
	model := sampleModel()
	updated, cmd := updateModel(t, model, tea.WindowSizeMsg{Width: 100, Height: 35})
	if updated.width != 100 || updated.height != 35 || cmd != nil {
		t.Fatal("window size was not stored")
	}

	weather := data.Weather{Condition: "Clear"}
	forecast := data.Forecast{Condition: "Rain"}
	overview := data.TodayOverview{}
	updated, _ = updateModel(t, updated, weatherMsg{today: weather, tomorrow: forecast, overview: overview})
	if updated.today.Condition != "Clear" || updated.overview.Holiday != model.overview.Holiday {
		t.Fatal("weather update failed to retain fallback holiday")
	}
	unchanged, _ := updateModel(t, updated, weatherMsg{today: data.Weather{Condition: "Bad"}, err: errors.New("offline")})
	if unchanged.today.Condition != "Clear" {
		t.Fatal("failed weather update replaced existing data")
	}

	updated, _ = updateModel(t, updated, githubMsg{days: data.MockContributions(), total: 44})
	updated, _ = updateModel(t, updated, rssMsg{news: []data.NewsItem{{Title: "Fresh"}}})
	updated, _ = updateModel(t, updated, fundsMsg{funds: data.MockFunds()})
	updated, _ = updateModel(t, updated, fundsMsg{funds: data.MockStocks(), stocks: true})
	if updated.githubTotal != 44 || updated.news[0].Title != "Fresh" || len(updated.funds) == 0 || len(updated.stocks) == 0 {
		t.Fatal("data messages were not applied")
	}

	updated.track = data.Track{Playing: true, ElapsedSec: 4, TotalSec: 5}
	tickTime := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	updated, cmd = updateModel(t, updated, tickMsg(tickTime))
	if !updated.now.Equal(tickTime) || updated.track.ElapsedSec != 5 || cmd == nil {
		t.Fatal("clock tick did not update time/playback or schedule next tick")
	}

	updated, cmd = updateModel(t, updated, marketRotateMsg(time.Now()))
	if !updated.marketSlide || updated.marketNext != 1 || cmd == nil {
		t.Fatal("market rotation did not start a slide")
	}
	for range marketSlideSteps {
		updated, cmd = updateModel(t, updated, marketFrameMsg(time.Now()))
	}
	if updated.marketSlide || updated.marketPage != 1 || cmd == nil {
		t.Fatal("market slide did not complete")
	}
	updated, _ = updateModel(t, updated, todayNewsMsg(time.Now()))
	updated, _ = updateModel(t, updated, weatherRotateMsg(time.Now()))
	if !updated.todayNewsPage || !updated.weatherTomorrow {
		t.Fatal("rotating cards did not toggle")
	}

	updated, cmd = updateModel(t, updated, spotifyTrackMsg{err: data.ErrSpotifyAuthorizationRequired})
	if updated.track.Title != "" || !strings.Contains(updated.spotifyErr, "connect Spotify") || cmd == nil {
		t.Fatal("Spotify authorization error was not presented")
	}
	updated, _ = updateModel(t, updated, spotifyTrackMsg{track: data.Track{Title: "Song", CoverURL: "cover-url"}})
	if updated.track.Title != "Song" || updated.cover != "" {
		t.Fatal("Spotify track was not stored or stale cover was retained")
	}
	updated, _ = updateModel(t, updated, coverLoadedMsg{url: "stale", cover: "bad"})
	updated, _ = updateModel(t, updated, coverLoadedMsg{url: "cover-url", cover: "good"})
	if updated.cover != "good" {
		t.Fatal("cover URL correlation failed")
	}
	updated, _ = updateModel(t, updated, configurationServerMsg{err: errors.New("port busy")})
	if updated.spotifyStatus() != "Configuration server: port busy" {
		t.Fatalf("status = %q", updated.spotifyStatus())
	}

	_, quit := updateModel(t, updated, tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl}))
	if quit == nil {
		t.Fatal("ctrl+c did not return quit command")
	}
}

func TestModelIntegrationRendersResponsiveDashboard(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CLOCKY_ENV_FILE", t.TempDir()+"/missing.env")
	model := NewModel()
	if model.Init() == nil || model.width != 80 || model.height != 24 || model.configWeb == nil {
		t.Fatal("new model is not initialized for the first render")
	}
	model.now = time.Date(2026, 9, 14, 12, 34, 56, 0, time.UTC)
	model.stocks, model.funds = data.MockStocks(), data.MockFunds()
	model.track = data.Track{Title: "Test Song", Artist: "Test Artist", TotalSec: 180}
	for _, size := range []struct{ width, height int }{{120, 40}, {70, 40}, {28, 12}} {
		model.width, model.height = size.width, size.height
		view := model.View()
		if !view.AltScreen || view.Content == "" {
			t.Fatalf("%dx%d view is empty or not in alternate screen", size.width, size.height)
		}
		if lipgloss.Width(view.Content) > size.width {
			t.Errorf("%dx%d view width = %d", size.width, size.height, lipgloss.Width(view.Content))
		}
	}

	model.width, model.height = 120, 40
	model.todayNewsPage = true
	if content := model.View().Content; !strings.Contains(content, "News") {
		t.Fatal("integration view did not render rotated news card")
	}
	progress := renderUpdateProgress(50, 8, updateProgress{Version: "v1.2.3", Downloaded: 50, Total: 100, ETA: "2s", Downloading: true})
	if !strings.Contains(progress, "50%") || !strings.Contains(progress, "ETA 2s") {
		t.Fatalf("update progress = %q", progress)
	}
}

func TestSettingsChangesTriggerAsyncRefreshCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	store := settings.New()
	model := sampleModel()
	model.settings = store
	store.SetWeather("Recife", "BR", "")
	updated, cmd := updateModel(t, model, tickMsg(time.Now()))
	if updated.weatherCity != "Recife" || cmd == nil {
		t.Fatal("weather settings change did not schedule refresh")
	}
	store.SetGitHub("octocat")
	updated, cmd = updateModel(t, updated, tickMsg(time.Now()))
	if updated.githubUser != "octocat" || cmd == nil {
		t.Fatal("GitHub settings change did not schedule refresh")
	}
	store.SetRSS([]string{"https://example.test/rss"}, 4)
	updated, cmd = updateModel(t, updated, tickMsg(time.Now()))
	if updated.newsLimit != 4 || cmd == nil {
		t.Fatal("RSS settings change did not schedule refresh")
	}
	store.SetMarketSymbols([]string{"MXRF11"}, []string{"VALE3"})
	updated, cmd = updateModel(t, updated, tickMsg(time.Now()))
	if !sameStrings(updated.fundSymbols, []string{"MXRF11"}) || !sameStrings(updated.stockSymbols, []string{"VALE3"}) || cmd == nil {
		t.Fatal("market settings change did not reset and refresh markets")
	}
}

func sampleModel() Model {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	return Model{
		now: now, today: data.MockToday(), tomorrow: data.MockTomorrow(), news: data.MockNews(),
		github: data.MockContributions(), overview: data.MockTodayOverview(), stocks: data.MockStocks(), funds: data.MockFunds(),
		lastRefresh: now, width: 80, height: 24,
	}
}

func updateModel(t *testing.T, model Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	updated, cmd := model.Update(msg)
	result, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}
	return result, cmd
}

func resetVisuals() {
	applyVisualSettings(settings.New().Get())
}
