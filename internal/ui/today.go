package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"clocky/internal/data"
)

func renderDaylight(width int, now time.Time, overview data.TodayOverview) string {
	prefix := "Daylight  " + overview.Sunrise + " "
	suffix := " " + overview.Sunset
	barWidth := width - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	if barWidth < 3 {
		prefix = "Sun " + overview.Sunrise + " "
		barWidth = width - lipgloss.Width(prefix) - lipgloss.Width(suffix)
	}
	if barWidth < 1 {
		return truncateLine("Sun "+overview.Sunrise+"–"+overview.Sunset, width)
	}

	parseMinutes := func(clock string) int {
		parsed, err := time.Parse("15:04", clock)
		if err != nil {
			return 0
		}
		return parsed.Hour()*60 + parsed.Minute()
	}
	start := parseMinutes(overview.Sunrise)
	end := parseMinutes(overview.Sunset)
	current := now.Hour()*60 + now.Minute()
	progress := 0
	if end > start {
		progress = min(barWidth, max(0, (current-start)*barWidth/(end-start)))
	}
	bar := sunIcon.Render(strings.Repeat("━", progress)) + dim.Render(strings.Repeat("─", barWidth-progress))
	return prefix + bar + suffix
}

func renderTodayBox(width, height int, now time.Time, overview data.TodayOverview) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	maxLines := height - boxStyle.GetVerticalFrameSize()
	if maxLines == 1 {
		line := fmt.Sprintf("Holiday %s · %dd · UV %d", overview.HolidayDate, overview.DaysUntil, overview.UVIndex)
		if innerWidth < 32 {
			line = fmt.Sprintf("Hol %s · UV%d", strings.ReplaceAll(overview.HolidayDate, " ", ""), overview.UVIndex)
		}
		return boxStyle.Width(width).Height(height).Render(truncateLine(line, innerWidth))
	}

	header := "Today · updated at " + overview.UpdatedAt.Format("15:04")
	if innerWidth < 26 {
		header = "Today · " + overview.UpdatedAt.Format("15:04")
	}
	holiday := fmt.Sprintf("Holiday   %s · %d days", overview.HolidayDate, overview.DaysUntil)
	if innerWidth < 26 {
		holiday = fmt.Sprintf("Holiday %s · %dd", strings.ReplaceAll(overview.HolidayDate, " ", ""), overview.DaysUntil)
	}
	lines := []string{dim.Render(truncateLine(header, innerWidth))}
	if maxLines == 2 {
		lines = append(lines, holiday)
	} else {
		lines = append(lines, renderDaylight(innerWidth, now, overview))
		if maxLines >= 3 {
			lines = append(lines, fmt.Sprintf("Min %d°C · Max %d°C", overview.TempLow, overview.TempHigh))
		}
		if maxLines >= 5 {
			lines = append(lines, fmt.Sprintf("UV index  %s · %s", value.Render(strconv.Itoa(overview.UVIndex)), overview.UVLevel))
		}
		if maxLines >= 6 && overview.RainProbability > 0 {
			rain := fmt.Sprintf("Rain      %d%%", overview.RainProbability)
			if overview.RainTime != "" {
				rain += " at " + overview.RainTime
			}
			lines = append(lines, rain)
		}
		lines = append(lines, holiday)
		if maxLines >= 7 {
			lines = append(lines, "Air       "+positive.Render(overview.AirQuality))
		}
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for index := 1; index < len(lines); index++ {
		lines[index] = truncateLine(lines[index], innerWidth)
	}
	return boxStyle.Width(width).Height(height).Render(strings.Join(lines, "\n"))
}
