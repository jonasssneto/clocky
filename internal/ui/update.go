package ui

import (
	"errors"
	"time"

	"clocky/internal/data"

	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.settings != nil {
			m.settings.SetViewport(msg.Width, msg.Height)
		}

	case tickMsg:
		m.now = time.Time(msg)
		if m.track.Playing && m.track.ElapsedSec < m.track.TotalSec {
			m.track.ElapsedSec++
		}
		return m, tickEvery()

	case refreshMsg:
		m.today, m.tomorrow, m.news, m.github = data.FetchAll()
		m.lastRefresh = time.Time(msg)
		return m, refreshEvery()

	case spotifyPollMsg:
		return m, fetchSpotifyTrack(m.spotify)

	case spotifyTrackMsg:
		if errors.Is(msg.err, data.ErrSpotifyAuthorizationRequired) {
			m.track = data.Track{}
			m.cover = ""
			m.spotifyErr = "Open the configuration page to connect Spotify"
			return m, pollSpotifyAfter()
		}
		if msg.err != nil {
			m.spotifyErr = msg.err.Error()
			return m, pollSpotifyAfter()
		}
		previousCoverURL := m.track.CoverURL
		m.track = msg.track
		m.spotifyErr = ""
		if m.track.CoverURL != "" && (m.track.CoverURL != previousCoverURL || m.cover == "") {
			m.cover = ""
			return m, tea.Batch(pollSpotifyAfter(), loadAndRenderCover(m.track.CoverURL))
		}
		if m.track.CoverURL == "" {
			m.cover = ""
		}
		return m, pollSpotifyAfter()

	case configurationServerMsg:
		if msg.err != nil {
			m.configErr = msg.err.Error()
		}

	case marketRotateMsg:
		if !m.marketSlide {
			m.marketNext = (m.marketPage + 1) % 2
			m.marketStep = 0
			m.marketSlide = true
			return m, marketFrameAfter()
		}

	case marketFrameMsg:
		if m.marketSlide {
			m.marketStep++
			if m.marketStep >= marketSlideSteps {
				m.marketPage = m.marketNext
				m.marketStep = 0
				m.marketSlide = false
				return m, rotateMarketAfter()
			}
			return m, marketFrameAfter()
		}

	case coverLoadedMsg:
		if msg.err == nil && msg.url == m.track.CoverURL {
			m.cover = msg.cover
		}

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}
