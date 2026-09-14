package ui

import (
	"context"
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"clocky/internal/configweb"
	"clocky/internal/data"
	"clocky/internal/imaging"
	"clocky/internal/settings"
	"clocky/internal/updater"
)

type tickMsg time.Time
type refreshMsg time.Time
type marketRotateMsg time.Time
type marketFrameMsg time.Time
type todayNewsMsg time.Time
type weatherRotateMsg time.Time
type marqueeMsg time.Time
type weatherMsg struct {
	today    data.Weather
	tomorrow data.Forecast
	overview data.TodayOverview
	err      error
}
type githubMsg struct {
	days  []data.ContributionDay
	total int
	err   error
}
type rssMsg struct {
	news []data.NewsItem
	err  error
}
type fundsMsg struct {
	funds  []data.MarketAsset
	stocks bool
	err    error
}
type coverLoadedMsg struct {
	url   string
	cover string
	err   error
}
type spotifyPollMsg time.Time
type spotifyTrackMsg struct {
	track data.Track
	err   error
}
type configurationServerMsg struct{ err error }
type updateRequestMsg configweb.UpdateRequest
type updateProgressMsg struct {
	version  string
	progress updater.Progress
}
type updateFinishedMsg struct {
	version string
	err     error
}

const (
	refreshInterval     = 10 * time.Minute
	marketPageInterval  = 4 * time.Second
	todayNewsInterval   = 60 * time.Second
	weatherInterval     = 30 * time.Second
	marqueeInterval     = 240 * time.Millisecond
	marketFrameDelay    = 45 * time.Millisecond
	marketSlideSteps    = 8
	spotifyPollInterval = 5 * time.Second
)

func configuredIntervals(store *settings.Store) settings.Snapshot {
	if store != nil {
		return store.Get()
	}
	return settings.New().Get()
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshEvery(store *settings.Store) tea.Cmd {
	delay := refreshInterval
	if value := configuredIntervals(store).RefreshMinutes; value > 0 {
		delay = time.Duration(value) * time.Minute
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func rotateMarketAfter(store *settings.Store) tea.Cmd {
	delay := marketPageInterval
	if value := configuredIntervals(store).MarketRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return marketRotateMsg(t) })
}

func marketFrameAfter(store *settings.Store) tea.Cmd {
	delay := marketFrameDelay
	if value := configuredIntervals(store).MarketFrameMilliseconds; value > 0 {
		delay = time.Duration(value) * time.Millisecond
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return marketFrameMsg(t) })
}

func todayNewsAfter(store *settings.Store) tea.Cmd {
	delay := todayNewsInterval
	if value := configuredIntervals(store).NewsRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return todayNewsMsg(t) })
}

func weatherAfter(store *settings.Store) tea.Cmd {
	delay := weatherInterval
	if value := configuredIntervals(store).WeatherRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return weatherRotateMsg(t) })
}

func marqueeAfter(store *settings.Store) tea.Cmd {
	delay := marqueeInterval
	if value := configuredIntervals(store).MarqueeMilliseconds; value > 0 {
		delay = time.Duration(value) * time.Millisecond
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return marqueeMsg(t) })
}

func fetchWeather(city, country, key string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		today, tomorrow, overview, err := data.FetchLiveWeather(ctx, city, country, key)
		return weatherMsg{today: today, tomorrow: tomorrow, overview: overview, err: err}
	}
}

func fetchGitHub(username string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		days, total, err := data.FetchGitHubContributions(ctx, username)
		return githubMsg{days: days, total: total, err: err}
	}
}

func fetchRSS(feeds []string, limit int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		news, err := data.FetchRSS(ctx, feeds, limit)
		return rssMsg{news: news, err: err}
	}
}

func fetchMarket(symbols []string, stocks bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		assets, err := data.FetchMarketFunds(ctx, symbols)
		return fundsMsg{funds: assets, stocks: stocks, err: err}
	}
}

func pollSpotifyAfter(store *settings.Store) tea.Cmd {
	delay := spotifyPollInterval
	if value := configuredIntervals(store).SpotifyPollSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return spotifyPollMsg(t) })
}

func fetchSpotifyTrack(client *data.SpotifyClient) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return spotifyTrackMsg{err: errors.New("Spotify is not configured")} //nolint:staticcheck // Spotify is a proper noun.
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		track, err := client.CurrentTrack(ctx)
		return spotifyTrackMsg{track: track, err: err}
	}
}

func serveConfiguration(server *configweb.Server) tea.Cmd {
	if server == nil {
		return nil
	}
	return func() tea.Msg {
		return configurationServerMsg{err: server.Serve(context.Background())}
	}
}

func loadAndRenderCover(url string) tea.Cmd {
	return func() tea.Msg {
		cover, err := imaging.DownloadAndRenderCover(url, spotifyCoverWidth, spotifyCoverHeight)
		return coverLoadedMsg{url: url, cover: cover, err: err}
	}
}

func waitUpdateRequest(requests <-chan configweb.UpdateRequest) tea.Cmd {
	return func() tea.Msg {
		return updateRequestMsg(<-requests)
	}
}

func waitUpdateEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-events }
}

func runUpdate(request configweb.UpdateRequest, events chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			releases, err := updater.ListReleases(ctx)
			if err != nil {
				events <- updateFinishedMsg{version: request.Version, err: err}
				return
			}
			var selected updater.Release
			for _, release := range releases {
				if request.Version == "latest" || release.TagName == request.Version {
					selected = release
					break
				}
			}
			if selected.TagName == "" {
				events <- updateFinishedMsg{version: request.Version, err: errors.New("selected release is not available")}
				return
			}
			asset, ok := updater.AssetForCurrentPlatform(selected)
			if !ok {
				events <- updateFinishedMsg{version: selected.TagName, err: errors.New("selected release has no compatible asset")}
				return
			}
			err = updater.DownloadAndInstall(ctx, asset, func(progress updater.Progress) {
				events <- updateProgressMsg{version: selected.TagName, progress: progress}
			})
			events <- updateFinishedMsg{version: selected.TagName, err: err}
		}()
		return nil
	}
}
