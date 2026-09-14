package settings

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Visual struct {
	Scale       int    `yaml:"scale"`
	BoxPadding  int    `yaml:"box_padding"`
	ColumnGap   int    `yaml:"column_gap"`
	ChartHeight int    `yaml:"chart_height"`
	BorderColor string `yaml:"border_color"`
	TextColor   string `yaml:"text_color"`
	DimColor    string `yaml:"dim_color"`
	ValueColor  string `yaml:"value_color"`
	AccentColor string `yaml:"accent_color"`
	Positive    string `yaml:"positive_color"`
	Negative    string `yaml:"negative_color"`
}

type Snapshot struct {
	Visual

	WeatherCity             string   `yaml:"weather_city"`
	WeatherCountry          string   `yaml:"weather_country"`
	WeatherKey              string   `yaml:"weather_key"`
	GitHubUser              string   `yaml:"github_user"`
	RSSFeeds                []string `yaml:"rss_feeds"`
	FundSymbols             []string `yaml:"fund_symbols"`
	FiiSymbols              []string `yaml:"fii_symbols"`
	StockSymbols            []string `yaml:"stock_symbols"`
	NewsLimit               int      `yaml:"news_limit"`
	RefreshMinutes          int      `yaml:"refresh_minutes"`
	NewsRotationSeconds     int      `yaml:"news_rotation_seconds"`
	WeatherRotationSeconds  int      `yaml:"weather_rotation_seconds"`
	MarketRotationSeconds   int      `yaml:"market_rotation_seconds"`
	MarketFrameMilliseconds int      `yaml:"market_frame_milliseconds"`
	MarqueeMilliseconds     int      `yaml:"marquee_milliseconds"`
	SpotifyPollSeconds      int      `yaml:"spotify_poll_seconds"`
	UpdatesEnabled          bool     `yaml:"updates_enabled"`
	UpdateVersion           string   `yaml:"update_version"`
	ViewportWidth           int      `yaml:"viewport_width"`
	ViewportHeight          int      `yaml:"viewport_height"`
}

type Store struct {
	Snapshot

	mu         sync.RWMutex
	configPath string
}

func New() *Store {
	snapshot := Snapshot{Visual: Visual{
		Scale: 100, BoxPadding: 1, ColumnGap: 1, ChartHeight: 2,
		BorderColor: "240", TextColor: "15", DimColor: "240", ValueColor: "228",
		AccentColor: "#1DB954", Positive: "#39d353", Negative: "#f85149",
	}, WeatherCity: "São Paulo", WeatherCountry: "BR", GitHubUser: "", NewsLimit: 3, RefreshMinutes: 10, NewsRotationSeconds: 60, WeatherRotationSeconds: 30, MarketRotationSeconds: 4, MarketFrameMilliseconds: 45, MarqueeMilliseconds: 240, SpotifyPollSeconds: 5, UpdatesEnabled: false, UpdateVersion: "latest", ViewportWidth: 80, ViewportHeight: 24}
	configPath := defaultConfigPath()
	if configPath != "" {
		if contents, err := os.ReadFile(configPath); err == nil {
			loaded := snapshot
			if err := yaml.Unmarshal(contents, &loaded); err == nil {
				snapshot = loaded
			}
		}
	}
	if len(snapshot.FiiSymbols) == 0 && len(snapshot.FundSymbols) > 0 {
		snapshot.FiiSymbols = append([]string(nil), snapshot.FundSymbols...)
	}
	return &Store{Snapshot: snapshot, configPath: configPath}
}

func (s *Store) Get() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.Snapshot
	snapshot.RSSFeeds = append([]string(nil), s.RSSFeeds...)
	snapshot.FundSymbols = append([]string(nil), s.FundSymbols...)
	snapshot.FiiSymbols = append([]string(nil), s.FiiSymbols...)
	snapshot.StockSymbols = append([]string(nil), s.StockSymbols...)
	return snapshot
}

func (s *Store) SetVisual(visual Visual) {
	s.mu.Lock()
	s.Visual = visual
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetWeather(city, country, key string) {
	s.mu.Lock()
	s.WeatherCity, s.WeatherCountry, s.WeatherKey = city, country, key
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetGitHub(user string) {
	s.mu.Lock()
	s.GitHubUser = user
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetRSS(feeds []string, limit int) {
	s.mu.Lock()
	s.RSSFeeds = append([]string(nil), feeds...)
	s.NewsLimit = limit
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetFunds(symbols []string) {
	s.mu.Lock()
	s.FundSymbols = append([]string(nil), symbols...)
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetMarketSymbols(fiIs, stocks []string) {
	s.mu.Lock()
	s.FiiSymbols = append([]string(nil), fiIs...)
	s.StockSymbols = append([]string(nil), stocks...)
	s.FundSymbols = append([]string(nil), fiIs...)
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetIntervals(refreshMinutes, newsRotationSeconds, weatherRotationSeconds, marketRotationSeconds, marketFrameMilliseconds, marqueeMilliseconds, spotifyPollSeconds int) {
	s.mu.Lock()
	s.RefreshMinutes = refreshMinutes
	s.NewsRotationSeconds = newsRotationSeconds
	s.WeatherRotationSeconds = weatherRotationSeconds
	s.MarketRotationSeconds = marketRotationSeconds
	s.MarketFrameMilliseconds = marketFrameMilliseconds
	s.MarqueeMilliseconds = marqueeMilliseconds
	s.SpotifyPollSeconds = spotifyPollSeconds
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetUpdates(enabled bool, version string) {
	s.mu.Lock()
	s.UpdatesEnabled = enabled
	if version == "" {
		version = "latest"
	}
	s.UpdateVersion = version
	s.saveLocked()
	s.mu.Unlock()
}

func (s *Store) SetViewport(width, height int) {
	s.mu.Lock()
	s.ViewportWidth, s.ViewportHeight = width, height
	s.saveLocked()
	s.mu.Unlock()
}

func defaultConfigPath() string {
	configDirectory, err := os.UserConfigDir()
	if err != nil || configDirectory == "" {
		return ""
	}
	return filepath.Join(configDirectory, "clocky", "config.yaml")
}

func (s *Store) saveLocked() {
	if s.configPath == "" {
		return
	}
	contents, err := yaml.Marshal(s.Snapshot)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(s.configPath), 0o700); err != nil {
		return
	}
	temporaryPath := s.configPath + ".tmp"
	if err := os.WriteFile(temporaryPath, contents, 0o600); err != nil {
		return
	}
	if err := os.Rename(temporaryPath, s.configPath); err != nil {
		_ = os.Remove(temporaryPath)
	}
}
