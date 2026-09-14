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
	innerHeight := max(1, height-boxStyle.GetVerticalFrameSize())
	if len(days) >= 30*7 && innerWidth >= 30 && innerHeight >= 8 {
		return renderMonthlyGithub(width, height, lastRefresh, days)
	}
	displayDays := days
	if len(displayDays) > 7 {
		displayDays = displayDays[len(displayDays)-7:]
	}
	var labels, cells, compactCells strings.Builder
	total := 0
	for index, day := range displayDays {
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

func renderMonthlyGithub(width, height int, lastRefresh time.Time, days []data.ContributionDay) string {
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(1, height-boxStyle.GetVerticalFrameSize())
	lines := []string{dim.Render(truncateLine("GitHub · "+lastRefresh.Format("15:04")+" · year 991 · month 30 days", innerWidth))}
	for row := 0; row < 7 && len(lines) < innerHeight; row++ {
		var line strings.Builder
		for column := 0; column < 30; column++ {
			day := days[column*7+row]
			line.WriteString(heatLevels[min(day.Level, len(heatLevels)-1)].Render("■"))
		}
		lines = append(lines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, line.String()))
	}
	if len(lines) < innerHeight {
		lines = append(lines, dim.Render("less ")+heatLevels[0].Render("■")+" "+heatLevels[2].Render("■")+" "+heatLevels[4].Render("■")+dim.Render(" more"))
	}
	return boxStyle.Width(width).Height(height).Render(strings.Join(lines, "\n"))
}
