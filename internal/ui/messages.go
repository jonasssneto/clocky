package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"clocky/internal/imaging"
)

type tickMsg time.Time
type refreshMsg time.Time
type marketRotateMsg time.Time
type marketFrameMsg time.Time
type coverLoadedMsg struct {
	url   string
	cover string
	err   error
}

const (
	refreshInterval    = 5 * time.Minute
	marketPageInterval = 4 * time.Second
	marketFrameDelay   = 45 * time.Millisecond
	marketSlideSteps   = 8
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

func loadAndRenderCover(url string) tea.Cmd {
	return func() tea.Msg {
		cover, err := imaging.DownloadAndRenderCover(url, spotifyCoverWidth, spotifyCoverHeight)
		return coverLoadedMsg{url: url, cover: cover, err: err}
	}
}
