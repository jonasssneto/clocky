package ui

import (
	"charm.land/lipgloss/v2"
	"clocky/internal/settings"
)

var (
	visualColumnGap   = 1
	visualChartHeight = 2
	visualScale       = 100
)

var (
	dim        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	clockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	value      = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	sunIcon    = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	spotify    = lipgloss.NewStyle().Foreground(lipgloss.Color("#1DB954"))
	positive   = lipgloss.NewStyle().Foreground(lipgloss.Color("#39d353"))
	negative   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
	heatLevels = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("237")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#0e4429")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#006d32")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#26a641")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#39d353")),
	}

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)
)

func applyVisualSettings(snapshot settings.Snapshot) {
	visual := snapshot.Visual
	visualColumnGap = max(0, min(4, visual.ColumnGap))
	visualChartHeight = max(1, min(4, visual.ChartHeight))
	visualScale = max(50, min(100, visual.Scale))
	dim = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.DimColor))
	clockStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.TextColor)).Bold(true)
	value = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.ValueColor))
	sunIcon = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.AccentColor))
	spotify = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.AccentColor))
	positive = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.Positive))
	negative = lipgloss.NewStyle().Foreground(lipgloss.Color(visual.Negative))
	boxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(visual.BorderColor)).
		Padding(0, max(0, min(2, visual.BoxPadding)))
	for index, color := range []string{visual.DimColor, "#0e4429", "#006d32", "#26a641", visual.Positive} {
		heatLevels[index] = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
}
