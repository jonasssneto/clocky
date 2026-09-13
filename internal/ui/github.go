package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
)

func renderGithubBox(width, height int, lastRefresh time.Time, days []data.ContributionDay) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	var labels, cells, compactCells strings.Builder
	total := 0
	for index, day := range days {
		if index > 0 {
			labels.WriteByte(' ')
			cells.WriteString("   ")
			compactCells.WriteByte(' ')
		}
		labels.WriteString(day.Label)
		cell := heatLevels[min(day.Level, len(heatLevels)-1)].Render("■")
		cells.WriteString(cell)
		compactCells.WriteString(cell)
		total += day.Count
	}

	header := "GitHub · updated at " + lastRefresh.Format("15:04")
	if lipgloss.Width(header) > innerWidth {
		header = "GitHub · " + lastRefresh.Format("15:04")
	}
	if lipgloss.Width(header) > innerWidth {
		header = "Git " + lastRefresh.Format("15:04")
	}

	summary := fmt.Sprintf("contributions last year 991 · last week %d", total)
	var lines []string
	if innerWidth >= max(lipgloss.Width(labels.String()), lipgloss.Width(summary)) {
		lines = []string{
			dim.Render(header),
			summary,
			labels.String(),
			cells.String(),
			dim.Render("less ") + heatLevels[0].Render("■") + " " + heatLevels[2].Render("■") + " " + heatLevels[4].Render("■") + dim.Render(" more"),
		}
	} else {
		compactSummary := fmt.Sprintf("year 991 · week %d", total)
		if lipgloss.Width(compactSummary) > innerWidth {
			compactSummary = fmt.Sprintf("991 · %d", total)
		}
		lines = []string{dim.Render(header), compactSummary, compactCells.String(), dim.Render("■ ■ ■")}
	}
	maxLines := height - boxStyle.GetVerticalFrameSize()
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return boxStyle.Width(width).Height(height).Render(strings.Join(lines, "\n"))
}
