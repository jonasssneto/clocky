// Package updater reads Clocky releases published on GitHub.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releasesURL           = "https://api.github.com/repos/jonasssneto/clocky/releases"
	releaseRequestTimeout = 10 * time.Second
	assetDownloadTimeout  = 10 * time.Minute
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

type Progress struct {
	Downloaded int64
	Total      int64
}

func DownloadAndInstall(ctx context.Context, asset Asset, report func(Progress)) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.URL, nil)
	if err != nil {
		return fmt.Errorf("prepare Clocky update download: %w", err)
	}
	request.Header.Set("Accept", "application/octet-stream")
	request.Header.Set("User-Agent", "clocky-updater")
	client := &http.Client{Timeout: assetDownloadTimeout}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download Clocky update: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("download Clocky update: GitHub returned %s", response.Status)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate Clocky executable: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(executable), ".clocky-update-*")
	if err != nil {
		return fmt.Errorf("create Clocky update file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o755); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("prepare Clocky update file: %w", err)
	}
	if err := copyAsset(response, temporary, report); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close Clocky update file: %w", err)
	}
	if err := os.Rename(temporaryPath, executable); err != nil {
		return fmt.Errorf("install Clocky update: %w", err)
	}
	return nil
}

func copyAsset(response *http.Response, destination io.Writer, report func(Progress)) error {
	reader := &progressReader{reader: response.Body, total: response.ContentLength, report: report}
	if _, err := io.Copy(destination, reader); err != nil {
		return fmt.Errorf("write Clocky update: %w", err)
	}
	return nil
}

type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	report     func(Progress)
}

func (r *progressReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	r.downloaded += int64(count)
	if r.report != nil {
		r.report(Progress{Downloaded: r.downloaded, Total: r.total})
	}
	return count, err
}
