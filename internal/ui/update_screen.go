package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// updateProgress contains the display-only state for an OTA update.
type updateProgress struct {
	Version     string
	Downloaded  int64
	Total       int64
	ETA         string
	Status      string
	Downloading bool
}

func renderUpdateProgress(width, height int, progress updateProgress) string {
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	barWidth := max(10, innerWidth-2)
	percent := 0
	if progress.Total > 0 {
		percent = min(100, max(0, int(progress.Downloaded*100/progress.Total)))
	}
	filled := barWidth * percent / 100
	bar := strings.Repeat("━", filled) + strings.Repeat("─", barWidth-filled)
	barLine := "  " + bar
	percentLabel := fmt.Sprintf("%d%%", percent)
	if lipgloss.Width(barLine)+1+lipgloss.Width(percentLabel) <= innerWidth {
		barLine += strings.Repeat(" ", innerWidth-lipgloss.Width(barLine)-lipgloss.Width(percentLabel)) + percentLabel
	} else {
		barLine = truncateLine(barLine+" "+percentLabel, innerWidth)
	}
	eta := strings.TrimSpace(progress.ETA)
	etaLine := ""
	if eta != "" {
		etaLine = strings.Repeat(" ", max(0, innerWidth-lipgloss.Width("ETA "+eta))) + "ETA " + eta
	}
	lines := []string{"Downloading " + truncateLine(progress.Version, max(1, innerWidth-11)), "", spotify.Render(barLine)}
	if etaLine != "" {
		lines = append(lines, etaLine)
	}
	content := lipgloss.NewStyle().Width(innerWidth).AlignHorizontal(lipgloss.Center).Render(strings.Join(lines, "\n"))
	return boxStyle.Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center).Render(content)
}
