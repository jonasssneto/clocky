package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
)

func renderGithubBox(width, height int, lastRefresh time.Time, days []data.ContributionDay, yearlyTotal int) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(1, height-boxStyle.GetVerticalFrameSize())
	if len(days) >= 30 && innerWidth >= 32 && innerHeight >= 4 {
		return renderMonthlyGithub(width, height, lastRefresh, days, yearlyTotal)
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

	if yearlyTotal == 0 {
		yearlyTotal = 991
	}
	summary := fmt.Sprintf("contributions last year %d · last week %d", yearlyTotal, total)
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

func renderMonthlyGithub(width, height int, lastRefresh time.Time, days []data.ContributionDay, yearlyTotal int) string {
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(1, height-boxStyle.GetVerticalFrameSize())
	if yearlyTotal == 0 {
		yearlyTotal = 991
	}
	lines := []string{dim.Render(truncateLine(fmt.Sprintf("GitHub · %s · year %d · last 30 days", lastRefresh.Format("15:04"), yearlyTotal), innerWidth))}
	lastMonth := days[len(days)-30:]
	var grid, labels strings.Builder
	for index, day := range lastMonth {
		if index > 0 && index%10 == 0 {
			grid.WriteByte(' ')
		}
		grid.WriteString(heatLevels[min(day.Level, len(heatLevels)-1)].Render("■"))
		if index == 0 || index == 9 || index == 19 || index == 29 {
			if labels.Len() > 0 {
				labels.WriteString("        ")
			}
			labels.WriteString(dayLabel(day.Label))
		}
	}
	if len(lines) < innerHeight {
		lines = append(lines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, labels.String()))
	}
	if len(lines) < innerHeight {
		lines = append(lines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, grid.String()))
	}
	if len(lines) < innerHeight {
		lines = append(lines, dim.Render("less ")+heatLevels[0].Render("■")+" "+heatLevels[2].Render("■")+" "+heatLevels[4].Render("■")+dim.Render(" more"))
	}
	return boxStyle.Width(width).Height(height).Render(strings.Join(lines, "\n"))
}

func dayLabel(date string) string {
	if len(date) >= 2 {
		return date[len(date)-2:]
	}
	return date
}
