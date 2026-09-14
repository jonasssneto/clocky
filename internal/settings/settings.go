package settings

import "sync"

type Visual struct {
	Scale       int
	BoxPadding  int
	ColumnGap   int
	ChartHeight int
	BorderColor string
	TextColor   string
	DimColor    string
	ValueColor  string
	AccentColor string
	Positive    string
	Negative    string
}

type Snapshot struct {
	Visual
	WeatherCity    string
	WeatherCountry string
	WeatherKey     string
	GitHubUser     string
	ViewportWidth  int
	ViewportHeight int
}

type Store struct {
	mu sync.RWMutex
	Snapshot
}

func New() *Store {
	return &Store{Snapshot: Snapshot{Visual: Visual{
		Scale: 100, BoxPadding: 1, ColumnGap: 1, ChartHeight: 2,
		BorderColor: "240", TextColor: "15", DimColor: "240", ValueColor: "228",
		AccentColor: "#1DB954", Positive: "#39d353", Negative: "#f85149",
	}, WeatherCity: "São Paulo", WeatherCountry: "BR", GitHubUser: "", ViewportWidth: 80, ViewportHeight: 24}}
}

func (s *Store) Get() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Snapshot
}

func (s *Store) SetVisual(visual Visual) {
	s.mu.Lock()
	s.Visual = visual
	s.mu.Unlock()
}

func (s *Store) SetWeather(city, country, key string) {
	s.mu.Lock()
	s.WeatherCity, s.WeatherCountry, s.WeatherKey = city, country, key
	s.mu.Unlock()
}

func (s *Store) SetGitHub(user string) {
	s.mu.Lock()
	s.GitHubUser = user
	s.mu.Unlock()
}

func (s *Store) SetViewport(width, height int) {
	s.mu.Lock()
	s.ViewportWidth, s.ViewportHeight = width, height
	s.mu.Unlock()
}
