package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type githubContributionResponse struct {
	Total         map[string]int `json:"total"`
	Contributions []struct {
		Date  string `json:"date"`
		Count int    `json:"count"`
		Level int    `json:"level"`
	} `json:"contributions"`
}

// FetchGitHubContributions uses the public, username-only contribution service.
// The service caches results for roughly one hour and does not require a token.
func FetchGitHubContributions(ctx context.Context, username string) ([]ContributionDay, int, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, 0, fmt.Errorf("GitHub profile is not configured")
	}
	endpoint := "https://github-contributions-api.jogruber.de/v4/" + url.PathEscape(username) + "?y=last"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, 0, fmt.Errorf("GitHub contributions API: HTTP status %d", response.StatusCode)
	}
	var payload githubContributionResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, 0, err
	}
	if len(payload.Contributions) == 0 {
		return nil, 0, fmt.Errorf("GitHub profile %q has no contribution data", username)
	}
	days := make([]ContributionDay, 0, len(payload.Contributions))
	for _, contribution := range payload.Contributions {
		days = append(days, ContributionDay{Label: contribution.Date, Count: contribution.Count, Level: max(0, min(4, contribution.Level))})
	}
	if len(days) > 30 {
		days = days[len(days)-30:]
	}
	total := payload.Total["lastYear"]
	if total == 0 {
		for _, value := range payload.Total {
			total += value
		}
	}
	return days, total, nil
}
