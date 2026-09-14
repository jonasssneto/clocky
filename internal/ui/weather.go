package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
)

func renderTodayColumn(weather data.Weather, title string) string {
	var rendered strings.Builder
	fmt.Fprintln(&rendered, dim.Render(title))
	lines := []string{
		fmt.Sprintf("%s  %s", weather.Condition, value.Render(fmt.Sprintf("%d°C", weather.TempC))),
		fmt.Sprintf("Feels like: %s", value.Render(fmt.Sprintf("%d°C", weather.FeelsLike))),
		fmt.Sprintf("Wind: %s", weather.Wind),
		fmt.Sprintf("Humidity: %d%%", weather.Humidity),
	}
	for index, icon := range weather.Icon {
		info := ""
		if index < len(lines) {
			info = lines[index]
		}
		fmt.Fprintf(&rendered, "%s  %s\n", sunIcon.Render(icon), info)
	}
	return strings.TrimRight(rendered.String(), "\n")
}

func renderTomorrowColumn(forecast data.Forecast) string {
	var rendered strings.Builder
	fmt.Fprintln(&rendered, dim.Render("Tomorrow"))
	lines := []string{
		forecast.Condition,
		fmt.Sprintf("High: %s", value.Render(fmt.Sprintf("%d°C", forecast.TempHigh))),
		fmt.Sprintf("Low: %s", value.Render(fmt.Sprintf("%d°C", forecast.TempLow))),
		"",
	}
	for index, icon := range forecast.Icon {
		info := ""
		if index < len(lines) {
			info = lines[index]
		}
		fmt.Fprintf(&rendered, "%s  %s\n", sunIcon.Render(icon), info)
	}
	return strings.TrimRight(rendered.String(), "\n")
}

func renderCompactWeather(weather data.Weather, cardWidth int, city string) string {
	innerWidth := max(1, cardWidth-boxStyle.GetHorizontalFrameSize())
	lines := strings.Split(renderTodayColumn(weather, weatherTitle(city)), "\n")
	for index, line := range lines {
		lines[index] = truncateLine(line, innerWidth)
	}
	return strings.Join(lines, "\n")
}

func renderWeather(weather data.Weather, forecast data.Forecast, cardWidth int, city string) string {
	today := renderTodayColumn(weather, weatherTitle(city))
	tomorrow := renderTomorrowColumn(forecast)
	innerWidth := max(1, cardWidth-boxStyle.GetHorizontalFrameSize())
	horizontal := lipgloss.JoinHorizontal(lipgloss.Top, today, dim.Render("  │  "), tomorrow)
	if lipgloss.Width(horizontal) <= innerWidth {
		return horizontal
	}
	return renderCompactWeather(weather, cardWidth, city)
}

func weatherTitle(city string) string {
	city = strings.TrimSpace(city)
	if city == "" {
		city = "São Paulo"
	}
	return "Weather · " + city
}
