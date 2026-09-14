package ui

import (
	"strings"
	"sync"

	"github.com/charmbracelet/x/ansi"
)

var marqueeState struct {
	sync.RWMutex
	offset int
}

func truncateLine(text string, width int) string {
	return fitMarquee(text, width)
}

func advanceMarquee() {
	marqueeState.Lock()
	marqueeState.offset++
	marqueeState.Unlock()
}

func fitMarquee(text string, width int) string {
	if width <= 0 || ansi.StringWidth(text) <= width {
		return ansi.Truncate(strings.TrimSpace(text), width, "…")
	}
	text = strings.TrimSpace(text)
	separator := "   "
	totalWidth := ansi.StringWidth(text) + ansi.StringWidth(separator)
	if totalWidth <= 0 {
		return ""
	}
	marqueeState.RLock()
	offset := marqueeState.offset % totalWidth
	marqueeState.RUnlock()
	cycle := text + separator + text
	return ansi.Cut(cycle, offset, offset+width)
}
