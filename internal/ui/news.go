package ui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
)

func renderNewsBox(width, height int, lastRefresh time.Time, news []data.NewsItem, limit int) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	style := boxStyle.Width(width)
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	maxLines := height - boxStyle.GetVerticalFrameSize()
	header := truncateLine("News · updated at "+lastRefresh.Format("15:04"), innerWidth)
	lines := []string{dim.Render(header)}
	for index, item := range news {
		if index >= limit || len(lines)+1 > maxLines {
			break
		}
		prefix := "▸ "
		source := " — " + item.Source
		titleWidth := innerWidth - lipgloss.Width(prefix) - lipgloss.Width(source)
		if titleWidth < 8 {
			source = ""
			titleWidth = innerWidth - lipgloss.Width(prefix)
		}
		wrapped := wrapNewsTitle(item.Title, titleWidth)
		for lineIndex, title := range wrapped {
			if len(lines) >= maxLines {
				break
			}
			line := title
			if lineIndex == 0 {
				line = prefix + line
			}
			if lineIndex == len(wrapped)-1 {
				line += dim.Render(source)
			}
			lines = append(lines, line)
		}
	}
	return style.Height(height).Render(strings.Join(lines, "\n"))
}

func wrapNewsTitle(title string, width int) []string {
	width = max(1, width)
	words := strings.Fields(title)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		for len([]rune(word)) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			wordRunes := []rune(word)
			lines = append(lines, string(wordRunes[:width]))
			word = string(wordRunes[width:])
		}
		if word == "" {
			continue
		}
		if current == "" {
			current = word
		} else if len(current)+1+len(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
