package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreDefaultsPersistenceAndDefensiveCopies(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	store := New()
	defaults := store.Get()
	if defaults.Scale != 100 || defaults.WeatherCity != "São Paulo" || defaults.NewsLimit != 3 {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	visual := defaults.Visual
	visual.Scale = 80
	store.SetVisual(visual)
	store.SetWeather("Curitiba", "BR", "key")
	store.SetGitHub("octocat")
	store.SetRSS([]string{"https://example.test/rss"}, 5)
	store.SetFunds([]string{"OLD11"})
	store.SetMarketSymbols([]string{"MXRF11"}, []string{"VALE3"})
	store.SetIntervals(15, 20, 25, 30, 35, 40, 45)
	store.SetUpdates(true, "")
	store.SetViewport(120, 40)

	wantPath := filepath.Join(configHome, "clocky", "config.yaml")
	if info, err := os.Stat(wantPath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("saved config = %v, mode %v", err, func() os.FileMode {
			if info == nil {
				return 0
			}
			return info.Mode().Perm()
		}())
	}
	reloaded := New().Get()
	if reloaded.Scale != 80 || reloaded.WeatherCity != "Curitiba" || reloaded.GitHubUser != "octocat" || reloaded.UpdateVersion != "latest" || reloaded.ViewportWidth != 120 {
		t.Fatalf("persisted snapshot = %+v", reloaded)
	}
	if !reflect.DeepEqual(reloaded.FiiSymbols, []string{"MXRF11"}) || !reflect.DeepEqual(reloaded.StockSymbols, []string{"VALE3"}) {
		t.Fatalf("persisted market symbols = %v, %v", reloaded.FiiSymbols, reloaded.StockSymbols)
	}

	snapshotCopy := store.Get()
	snapshotCopy.RSSFeeds[0] = "changed"
	snapshotCopy.FiiSymbols[0] = "CHANGED"
	if store.Get().RSSFeeds[0] == "changed" || store.Get().FiiSymbols[0] == "CHANGED" {
		t.Fatal("Get exposes mutable slices")
	}
}

func TestStoreLoadsLegacyFundSymbolsAndIgnoresInvalidYAML(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "clocky", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fund_symbols: [LEGACY11]\nweather_city: Recife\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot := New().Get()
	if !reflect.DeepEqual(snapshot.FiiSymbols, []string{"LEGACY11"}) || snapshot.WeatherCity != "Recife" {
		t.Fatalf("legacy config = %+v", snapshot)
	}
	if err := os.WriteFile(path, []byte("[invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := New().Get(); got.Scale != 100 || got.WeatherCity != "São Paulo" {
		t.Fatalf("invalid YAML should preserve defaults: %+v", got)
	}
}
