package data

import "testing"

func TestFormatClockAcceptsOpenMeteoLocalTime(t *testing.T) {
	if got := formatClock([]string{"2026-09-13T06:03"}); got != "06:03" {
		t.Fatalf("formatClock() = %q, want 06:03", got)
	}
	if got := formatClock([]string{"2026-09-13T18:07:00-03:00"}); got != "18:07" {
		t.Fatalf("formatClock() = %q, want 18:07", got)
	}
}
