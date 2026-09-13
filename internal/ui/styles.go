package ui

import "charm.land/lipgloss/v2"

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
