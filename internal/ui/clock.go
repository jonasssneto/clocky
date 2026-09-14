package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

var bigFont = map[rune][]string{
	'0': {"███", "█ █", "█ █", "█ █", "███"},
	'1': {" █ ", "██ ", " █ ", " █ ", "███"},
	'2': {"███", "  █", "███", "█  ", "███"},
	'3': {"███", "  █", " ██", "  █", "███"},
	'4': {"█ █", "█ █", "███", "  █", "  █"},
	'5': {"███", "█  ", "███", "  █", "███"},
	'6': {"███", "█  ", "███", "█ █", "███"},
	'7': {"███", "  █", "  █", " █ ", " █ "},
	'8': {"███", "█ █", "███", "█ █", "███"},
	'9': {"███", "█ █", "███", "  █", "███"},
	':': {"   ", " █ ", "   ", " █ ", "   "},
}

func bigText(text string) []string {
	rows := make([]string, 5)
	for _, character := range text {
		glyph, ok := bigFont[character]
		if !ok {
			continue
		}
		for row := 0; row < 5; row++ {
			rows[row] += glyph[row] + " "
		}
	}
	return rows
}

func renderLargeClock(now time.Time) string {
	var rendered strings.Builder
	for _, row := range bigText(now.Format("15:04:05")) {
		fmt.Fprintln(&rendered, clockStyle.Render(row))
	}
	fmt.Fprint(&rendered, dim.Render(now.Format("Mon, 02 Jan 2006")))
	return rendered.String()
}

func renderClock(now time.Time, cardWidth int) string {
	largeClock := renderLargeClock(now)
	innerWidth := max(1, cardWidth-boxStyle.GetHorizontalFrameSize())
	if visualScale >= 100 && lipgloss.Width(largeClock) <= innerWidth {
		return largeClock
	}
	return lipgloss.JoinVertical(
		lipgloss.Center,
		clockStyle.Render(now.Format("15:04:05")),
		dim.Render(now.Format("Mon, 02 Jan 2006")),
	)
}
