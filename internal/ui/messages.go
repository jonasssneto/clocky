package ui

import (
	"context"
	"errors"
	"time"

	"clocky/internal/configweb"
	"clocky/internal/data"
	"clocky/internal/imaging"

	tea "charm.land/bubbletea/v2"
)

type tickMsg time.Time
type refreshMsg time.Time
type marketRotateMsg time.Time
type marketFrameMsg time.Time
type todayNewsMsg time.Time
type weatherMsg struct {
	today    data.Weather
	tomorrow data.Forecast
	overview data.TodayOverview
	err      error
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
	marketFrameDelay    = 45 * time.Millisecond
	marketSlideSteps    = 8
	spotifyPollInterval = 5 * time.Second
)

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func refreshEvery() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return refreshMsg(t) })
}

func rotateMarketAfter() tea.Cmd {
	return tea.Tick(marketPageInterval, func(t time.Time) tea.Msg { return marketRotateMsg(t) })
}

func marketFrameAfter() tea.Cmd {
	return tea.Tick(marketFrameDelay, func(t time.Time) tea.Msg { return marketFrameMsg(t) })
}

func todayNewsAfter() tea.Cmd {
	return tea.Tick(todayNewsInterval, func(t time.Time) tea.Msg { return todayNewsMsg(t) })
}

func fetchWeather(city, country, key string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		today, tomorrow, overview, err := data.FetchLiveWeather(ctx, city, country, key)
		return weatherMsg{today: today, tomorrow: tomorrow, overview: overview, err: err}
	}
}

func pollSpotifyAfter() tea.Cmd {
	return tea.Tick(spotifyPollInterval, func(t time.Time) tea.Msg { return spotifyPollMsg(t) })
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
