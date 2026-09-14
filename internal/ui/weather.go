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

func renderTomorrowColumn(forecast data.Forecast, title string) string {
	var rendered strings.Builder
	fmt.Fprintln(&rendered, dim.Render(title))
	rain := fmt.Sprintf("Rain: %d%%", forecast.RainProbability)
	if forecast.RainTime != "" {
		rain += " at " + forecast.RainTime
	}
	lines := []string{
		forecast.Condition,
		fmt.Sprintf("High: %s", value.Render(fmt.Sprintf("%d°C", forecast.TempHigh))),
		fmt.Sprintf("Low: %s", value.Render(fmt.Sprintf("%d°C", forecast.TempLow))),
		rain,
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
	lines := strings.Split(renderTodayColumn(weather, weatherTitle(city, "Today")), "\n")
	for index, line := range lines {
		lines[index] = truncateLine(line, innerWidth)
	}
	return strings.Join(lines, "\n")
}

func renderWeather(weather data.Weather, forecast data.Forecast, cardWidth int, city string, showTomorrow bool) string {
	if showTomorrow {
		return renderTomorrowColumn(forecast, weatherTitle(city, "Tomorrow"))
	}
	today := renderTodayColumn(weather, weatherTitle(city, "Today"))
	innerWidth := max(1, cardWidth-boxStyle.GetHorizontalFrameSize())
	if lipgloss.Width(today) <= innerWidth {
		return today
	}
	return renderCompactWeather(weather, cardWidth, city)
}

func weatherTitle(city, period string) string {
	city = strings.TrimSpace(city)
	if city == "" {
		city = "São Paulo"
	}
	return "Weather · " + city + " · " + period
}
