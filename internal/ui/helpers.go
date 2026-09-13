package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func truncateLine(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(strings.TrimSpace(text), width, "…")
}
