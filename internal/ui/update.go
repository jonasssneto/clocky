package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"clocky/internal/data"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.now = time.Time(msg)
		return m, tickEvery()

	case refreshMsg:
		previousCoverURL := m.track.CoverURL
		m.today, m.tomorrow, m.news, m.github, m.track = data.FetchAll()
		m.lastRefresh = time.Time(msg)
		if m.track.CoverURL != previousCoverURL || m.cover == "" {
			m.cover = ""
			return m, tea.Batch(refreshEvery(), loadAndRenderCover(m.track.CoverURL))
		}
		return m, refreshEvery()

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
