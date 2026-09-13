package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"clocky/internal/data"
)

type Model struct {
	now         time.Time
	today       data.Weather
	tomorrow    data.Forecast
	news        []data.NewsItem
	github      []data.ContributionDay
	track       data.Track
	overview    data.TodayOverview
	stocks      []data.MarketAsset
	funds       []data.MarketAsset
	marketPage  int
	marketNext  int
	marketStep  int
	marketSlide bool
	cover       string
	lastRefresh time.Time
	width       int
	height      int
}

func NewModel() Model {
	today, tomorrow, news, github, track := data.FetchAll()
	return Model{
		now:         time.Now(),
		today:       today,
		tomorrow:    tomorrow,
		news:        news,
		github:      github,
		track:       track,
		overview:    data.MockTodayOverview(),
		stocks:      data.MockStocks(),
		funds:       data.MockFunds(),
		lastRefresh: time.Now(),
		width:       80,
		height:      24,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickEvery(), refreshEvery(), rotateMarketAfter(), loadAndRenderCover(m.track.CoverURL))
}
