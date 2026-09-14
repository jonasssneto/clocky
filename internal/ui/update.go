package ui

import (
	"errors"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"clocky/internal/configweb"
	"clocky/internal/data"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.settings != nil {
			m.settings.SetViewport(msg.Width, msg.Height)
		}

	case weatherMsg:
		if msg.err == nil {
			if msg.overview.Holiday == "" {
				msg.overview.Holiday = m.overview.Holiday
				msg.overview.HolidayDate = m.overview.HolidayDate
				msg.overview.DaysUntil = m.overview.DaysUntil
			}
			m.today, m.tomorrow, m.overview = msg.today, msg.tomorrow, msg.overview
		}

	case githubMsg:
		if msg.err == nil {
			m.github, m.githubTotal = msg.days, msg.total
		}

	case rssMsg:
		if msg.err == nil && len(msg.news) > 0 {
			m.news = msg.news
			m.lastRefresh = time.Now()
		}

	case fundsMsg:
		if msg.err == nil {
			if msg.stocks {
				m.stocks = msg.funds
			} else {
				m.funds = msg.funds
			}
		}

	case tickMsg:
		m.now = time.Time(msg)
		if m.track.Playing && m.track.ElapsedSec < m.track.TotalSec {
			m.track.ElapsedSec++
		}
		if m.settings != nil {
			snapshot := m.settings.Get()
			if snapshot.WeatherCity != m.weatherCity || snapshot.WeatherCountry != m.weatherCountry || snapshot.WeatherKey != m.weatherKey {
				m.weatherCity, m.weatherCountry, m.weatherKey = snapshot.WeatherCity, snapshot.WeatherCountry, snapshot.WeatherKey
				return m, tea.Batch(tickEvery(), fetchWeather(m.weatherCity, m.weatherCountry, m.weatherKey))
			}
			if snapshot.GitHubUser != m.githubUser {
				m.githubUser = snapshot.GitHubUser
				return m, tea.Batch(tickEvery(), fetchGitHub(m.githubUser))
			}
			if !sameStrings(snapshot.RSSFeeds, m.rssFeeds) || snapshot.NewsLimit != m.newsLimit {
				m.rssFeeds, m.newsLimit = append([]string(nil), snapshot.RSSFeeds...), snapshot.NewsLimit
				return m, tea.Batch(tickEvery(), fetchRSS(m.rssFeeds, m.newsLimit))
			}
			if !sameStrings(snapshot.FiiSymbols, m.fundSymbols) || !sameStrings(snapshot.StockSymbols, m.stockSymbols) {
				m.fundSymbols = append([]string(nil), snapshot.FiiSymbols...)
				m.stockSymbols = append([]string(nil), snapshot.StockSymbols...)
				m.marketPage, m.marketNext, m.marketStep = 0, 0, 0
				m.marketSlide = false
				return m, tea.Batch(tickEvery(), rotateMarketAfter(m.settings), fetchMarket(m.fundSymbols, false), fetchMarket(m.stockSymbols, true))
			}
		}
		return m, tickEvery()

	case refreshMsg:
		_, _, m.news, m.github = data.FetchAll()
		m.lastRefresh = time.Time(msg)
		return m, tea.Batch(refreshEvery(m.settings), fetchWeather(m.weatherCity, m.weatherCountry, m.weatherKey), fetchGitHub(m.githubUser), fetchRSS(m.rssFeeds, m.newsLimit), fetchMarket(m.fundSymbols, false), fetchMarket(m.stockSymbols, true))

	case spotifyPollMsg:
		return m, fetchSpotifyTrack(m.spotify)

	case spotifyTrackMsg:
		if errors.Is(msg.err, data.ErrSpotifyAuthorizationRequired) {
			m.track = data.Track{}
			m.cover = ""
			m.spotifyErr = "Open the configuration page to connect Spotify"
			return m, pollSpotifyAfter(m.settings)
		}
		if msg.err != nil {
			m.spotifyErr = msg.err.Error()
			return m, pollSpotifyAfter(m.settings)
		}
		previousCoverURL := m.track.CoverURL
		m.track = msg.track
		m.spotifyErr = ""
		if m.track.CoverURL != "" && (m.track.CoverURL != previousCoverURL || m.cover == "") {
			m.cover = ""
			return m, tea.Batch(pollSpotifyAfter(m.settings), loadAndRenderCover(m.track.CoverURL))
		}
		if m.track.CoverURL == "" {
			m.cover = ""
		}
		return m, pollSpotifyAfter(m.settings)

	case configurationServerMsg:
		if msg.err != nil {
			m.configErr = msg.err.Error()
		}

	case updateRequestMsg:
		if m.update != nil {
			return m, waitUpdateRequest(m.configWeb.UpdateRequests())
		}
		events := make(chan tea.Msg, 16)
		m.updateEvents = events
		m.update = &updateProgress{Version: msg.Version, Status: "Checking for the selected release…", Downloading: true}
		return m, tea.Batch(waitUpdateEvent(events), runUpdate(configweb.UpdateRequest(msg), events))

	case updateProgressMsg:
		if m.update != nil {
			m.update.Version = msg.version
			m.update.Downloaded = msg.progress.Downloaded
			m.update.Total = msg.progress.Total
			m.update.Status = "Downloading…"
		}
		return m, waitUpdateEvent(m.updateEvents)

	case updateFinishedMsg:
		if m.update != nil {
			m.update.Version = msg.version
			m.update.Downloading = false
			if msg.err != nil {
				m.update.Status = "Update failed: " + msg.err.Error()
			} else {
				m.update.Status = "Update installed. Restart Clocky to apply it."
			}
		}
		m.updateEvents = nil
		return m, waitUpdateRequest(m.configWeb.UpdateRequests())

	case marketRotateMsg:
		pageCount := max(len(m.stocks), len(m.funds))
		if pageCount <= 1 {
			return m, rotateMarketAfter(m.settings)
		}
		if m.marketSlide {
			return m, marketFrameAfter(m.settings)
		}
		if !m.marketSlide {
			m.marketNext = (m.marketPage + 1) % pageCount
			m.marketStep = 0
			m.marketSlide = true
			return m, marketFrameAfter(m.settings)
		}

	case marketFrameMsg:
		if m.marketSlide {
			m.marketStep++
			if m.marketStep >= marketSlideSteps {
				m.marketPage = m.marketNext
				m.marketStep = 0
				m.marketSlide = false
				return m, rotateMarketAfter(m.settings)
			}
			return m, marketFrameAfter(m.settings)
		}
		return m, rotateMarketAfter(m.settings)

	case todayNewsMsg:
		m.todayNewsPage = !m.todayNewsPage
		return m, todayNewsAfter(m.settings)

	case weatherRotateMsg:
		m.weatherTomorrow = !m.weatherTomorrow
		return m, weatherAfter(m.settings)

	case marqueeMsg:
		advanceMarquee()
		return m, marqueeAfter(m.settings)

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

func sameStrings(left, right []string) bool {
	return slices.Equal(left, right)
}
