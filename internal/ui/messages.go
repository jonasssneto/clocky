package ui

import (
	"context"
	"errors"
	"time"

	"clocky/internal/configweb"
	"clocky/internal/data"
	"clocky/internal/imaging"
	"clocky/internal/settings"

	tea "charm.land/bubbletea/v2"
)

type tickMsg time.Time
type refreshMsg time.Time
type marketRotateMsg time.Time
type marketFrameMsg time.Time
type todayNewsMsg time.Time
type weatherRotateMsg time.Time
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

const (
	refreshInterval     = 10 * time.Minute
	marketPageInterval  = 4 * time.Second
	todayNewsInterval   = 60 * time.Second
	weatherInterval     = 30 * time.Second
	marketFrameDelay    = 45 * time.Millisecond
	marketSlideSteps    = 8
	spotifyPollInterval = 5 * time.Second
)

var intervalStore *settings.Store

func configureIntervals(store *settings.Store) { intervalStore = store }

func configuredIntervals() settings.Snapshot {
	if intervalStore != nil {
		return intervalStore.Get()
	}
	return settings.New().Get()
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshEvery() tea.Cmd {
	delay := refreshInterval
	if value := configuredIntervals().RefreshMinutes; value > 0 {
		delay = time.Duration(value) * time.Minute
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func rotateMarketAfter() tea.Cmd {
	delay := marketPageInterval
	if value := configuredIntervals().MarketRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return marketRotateMsg(t) })
}

func marketFrameAfter() tea.Cmd {
	delay := marketFrameDelay
	if value := configuredIntervals().MarketFrameMilliseconds; value > 0 {
		delay = time.Duration(value) * time.Millisecond
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return marketFrameMsg(t) })
}

func todayNewsAfter() tea.Cmd {
	delay := todayNewsInterval
	if value := configuredIntervals().NewsRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return todayNewsMsg(t) })
}

func weatherAfter() tea.Cmd {
	delay := weatherInterval
	if value := configuredIntervals().WeatherRotationSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return weatherRotateMsg(t) })
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

func pollSpotifyAfter() tea.Cmd {
	delay := spotifyPollInterval
	if value := configuredIntervals().SpotifyPollSeconds; value > 0 {
		delay = time.Duration(value) * time.Second
	}
	return tea.Tick(delay, func(t time.Time) tea.Msg { return spotifyPollMsg(t) })
}

func fetchSpotifyTrack(client *data.SpotifyClient) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return spotifyTrackMsg{err: errors.New("Spotify is not configured")}
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
