package ui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
)

func renderNewsBox(width, height int, lastRefresh time.Time, news []data.NewsItem) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	style := boxStyle.Width(width)
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	maxLines := height - boxStyle.GetVerticalFrameSize()
	header := truncateLine("News · updated at "+lastRefresh.Format("15:04"), innerWidth)
	lines := []string{dim.Render(header)}
	for index, item := range news {
		if index >= 3 || len(lines)+1 > maxLines {
			break
		}
		prefix := "▸ "
		source := " — " + item.Source
		titleWidth := innerWidth - lipgloss.Width(prefix) - lipgloss.Width(source)
		if titleWidth < 8 {
			source = ""
			titleWidth = innerWidth - lipgloss.Width(prefix)
		}
		title := truncateLine(item.Title, titleWidth)
		lines = append(lines, prefix+title+dim.Render(source))
	}
	return style.Height(height).Render(strings.Join(lines, "\n"))
}
