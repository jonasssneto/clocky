// Package updater reads Clocky releases published on GitHub.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	releasesURL           = "https://api.github.com/repos/jonasssneto/clocky/releases"
	releaseRequestTimeout = 10 * time.Second
)

type Release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	PublishedAt string  `json:"published_at"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []Asset `json:"assets"`
}

type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func ListReleases(ctx context.Context) ([]Release, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL+"?per_page=30", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "clocky-updater")
	client := &http.Client{Timeout: releaseRequestTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("list Clocky releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			return []Release{}, nil
		}
		return nil, fmt.Errorf("list Clocky releases: GitHub returned %s", response.Status)
	}
	var releases []Release
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode Clocky releases: %w", err)
	}
	available := releases[:0]
	for _, release := range releases {
		if !release.Draft {
			available = append(available, release)
		}
	}
	return available, nil
}

func AssetForCurrentPlatform(release Release) (Asset, bool) {
	wanted := "clocky-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm" {
		wanted += "v7"
	}
	if runtime.GOOS == "windows" {
		wanted += ".exe"
	}
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, wanted) {
			return asset, true
		}
	}
	return Asset{}, false
}
