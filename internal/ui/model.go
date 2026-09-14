package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"clocky/internal/configweb"
	"clocky/internal/data"
	"clocky/internal/settings"
)

type Model struct {
	now             time.Time
	today           data.Weather
	tomorrow        data.Forecast
	news            []data.NewsItem
	rssFeeds        []string
	newsLimit       int
	github          []data.ContributionDay
	githubTotal     int
	track           data.Track
	overview        data.TodayOverview
	stocks          []data.MarketAsset
	funds           []data.MarketAsset
	fundSymbols     []string
	stockSymbols    []string
	marketPage      int
	marketNext      int
	marketStep      int
	marketSlide     bool
	todayNewsPage   bool
	weatherTomorrow bool
	cover           string
	spotify         *data.SpotifyClient
	configWeb       *configweb.Server
	settings        *settings.Store
	weatherCity     string
	weatherCountry  string
	weatherKey      string
	githubUser      string
	spotifyErr      string
	configErr       string
	spotifyPage     string
	lastRefresh     time.Time
	width           int
	height          int
}

func NewModel() Model {
	today, tomorrow, news, github := data.FetchAll()
	spotifyClient, spotifyErr := data.NewSpotifyClient()
	spotifyMessage := ""
	if spotifyErr != nil {
		spotifyMessage = spotifyErr.Error()
	}
	configurationURL := "http://127.0.0.1:8888/"
	visualSettings := settings.New()
	var configurationServer *configweb.Server
	var configurationErr error
	if spotifyClient != nil {
		configurationURL = spotifyClient.ConfigurationPageURL()
		configurationServer, configurationErr = configweb.New(configurationURL, spotifyClient)
	} else {
		configurationServer, configurationErr = configweb.New(configurationURL)
	}
	if configurationErr != nil {
		spotifyMessage = configurationErr.Error()
	}
	if configurationServer != nil {
		configurationServer.SetSettings(visualSettings)
	}
	configureIntervals(visualSettings)
	return Model{
		now:            time.Now(),
		today:          today,
		tomorrow:       tomorrow,
		news:           news,
		github:         github,
		track:          data.Track{},
		overview:       data.MockTodayOverview(),
		fundSymbols:    append([]string(nil), visualSettings.Get().FiiSymbols...),
		stockSymbols:   append([]string(nil), visualSettings.Get().StockSymbols...),
		spotify:        spotifyClient,
		configWeb:      configurationServer,
		settings:       visualSettings,
		weatherCity:    visualSettings.Get().WeatherCity,
		weatherCountry: visualSettings.Get().WeatherCountry,
		weatherKey:     visualSettings.Get().WeatherKey,
		githubUser:     visualSettings.Get().GitHubUser,
		githubTotal:    991,
		rssFeeds:       append([]string(nil), visualSettings.Get().RSSFeeds...),
		newsLimit:      visualSettings.Get().NewsLimit,
		spotifyErr:     spotifyMessage,
		configErr:      errorMessage(configurationErr),
		spotifyPage:    configurationURL,
		lastRefresh:    time.Now(),
		width:          80,
		height:         24,
	}
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickEvery(), marqueeAfter(), refreshEvery(), rotateMarketAfter(), todayNewsAfter(), weatherAfter(), fetchWeather(m.weatherCity, m.weatherCountry, m.weatherKey), fetchGitHub(m.githubUser), fetchRSS(m.rssFeeds, m.newsLimit), fetchMarket(m.fundSymbols, false), fetchMarket(m.stockSymbols, true), fetchSpotifyTrack(m.spotify), serveConfiguration(m.configWeb))
}
