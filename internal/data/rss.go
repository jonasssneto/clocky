package data

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type rssDocument struct {
	Channel rssChannel  `xml:"channel"`
	Entries []atomEntry `xml:"entry"`
	Title   string      `xml:"title"`
}
type rssChannel struct {
	Title string    `xml:"title"`
	Items []rssItem `xml:"item"`
}
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Date        string `xml:"date"`
}
type atomEntry struct {
	Title     string     `xml:"title"`
	Links     []atomLink `xml:"link"`
	Summary   string     `xml:"summary"`
	Updated   string     `xml:"updated"`
	Published string     `xml:"published"`
}
type atomLink struct {
	Href string `xml:"href,attr"`
}

type rssResult struct {
	item NewsItem
	date time.Time
}

// FetchRSS returns RSS items ordered by date. A single feed contributes up to
// limit items; when multiple feeds are configured, only the newest item from
// each feed is considered before applying the overall limit.
func FetchRSS(ctx context.Context, feeds []string, limit int) ([]NewsItem, error) {
	if len(feeds) == 0 {
		return nil, nil
	}
	client := &http.Client{Timeout: 10 * time.Second}
	results := make([]rssResult, 0, len(feeds)*maxInt(limit, 1))
	var failures int
	for _, feedURL := range feeds {
		feedResults, err := fetchRSSFeed(ctx, client, feedURL)
		if err != nil {
			failures++
			continue
		}
		if len(feeds) == 1 {
			results = append(results, feedResults...)
		} else if len(feedResults) > 0 {
			results = append(results, feedResults[0])
		}
	}
	if len(results) == 0 && failures > 0 {
		return nil, errors.New("all RSS feeds failed")
	}
	sort.SliceStable(results, func(left, right int) bool { return results[left].date.After(results[right].date) })
	if limit < 1 {
		limit = 1
	}
	if len(results) > limit {
		results = results[:limit]
	}
	items := make([]NewsItem, 0, len(results))
	for _, result := range results {
		items = append(items, result.item)
	}
	return items, nil
}

func fetchRSSFeed(ctx context.Context, client *http.Client, feedURL string) ([]rssResult, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Clocky RSS reader")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("RSS HTTP status %d", response.StatusCode)
	}
	var document rssDocument
	if err := xml.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&document); err != nil {
		return nil, err
	}
	results := make([]rssResult, 0)
	if len(document.Channel.Items) > 0 {
		for _, item := range document.Channel.Items {
			results = append(results, rssResult{
				item: NewsItem{Title: strings.TrimSpace(item.Title), Source: feedSource(feedURL, document.Channel.Title), Summary: strings.TrimSpace(item.Description)},
				date: parseFeedDate(firstNonEmpty(item.PubDate, item.Date)),
			})
		}
	}
	if len(results) == 0 && len(document.Entries) > 0 {
		for _, entry := range document.Entries {
			results = append(results, rssResult{
				item: NewsItem{Title: strings.TrimSpace(entry.Title), Source: feedSource(feedURL, document.Title), Summary: strings.TrimSpace(entry.Summary)},
				date: parseFeedDate(firstNonEmpty(entry.Published, entry.Updated)),
			})
		}
	}
	if len(results) == 0 {
		return nil, errors.New("RSS feed has no items")
	}
	sort.SliceStable(results, func(left, right int) bool { return results[left].date.After(results[right].date) })
	return results, nil
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseFeedDate(value string) time.Time {
	for _, layout := range []string{time.RFC3339, time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, "2006-01-02 15:04:05 -0700"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func feedSource(feedURL, title string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	parsed, err := url.Parse(feedURL)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return "RSS"
}
