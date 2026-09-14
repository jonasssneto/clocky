package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type geoLocation struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
}

type geocodingResponse struct {
	Results []geoLocation `json:"results"`
}

type forecastResponse struct {
	Current struct {
		Temperature   float64 `json:"temperature_2m"`
		FeelsLike     float64 `json:"apparent_temperature"`
		Humidity      float64 `json:"relative_humidity_2m"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
		WeatherCode   int     `json:"weather_code"`
	} `json:"current"`
	Daily struct {
		WeatherCode []int     `json:"weather_code"`
		TempMax     []float64 `json:"temperature_2m_max"`
		TempMin     []float64 `json:"temperature_2m_min"`
		UVMax       []float64 `json:"uv_index_max"`
		Sunrise     []string  `json:"sunrise"`
		Sunset      []string  `json:"sunset"`
	} `json:"daily"`
	Hourly struct {
		Time                     []string  `json:"time"`
		PrecipitationProbability []int     `json:"precipitation_probability"`
		Precipitation            []float64 `json:"precipitation"`
	} `json:"hourly"`
}

type airQualityResponse struct {
	Current struct {
		USAQI float64 `json:"us_aqi"`
	} `json:"current"`
}

type publicHoliday struct {
	Date      string `json:"date"`
	LocalName string `json:"localName"`
	Name      string `json:"name"`
}

func FetchLiveWeather(ctx context.Context, city, country, apiKey string) (Weather, Forecast, TodayOverview, error) {
	_ = apiKey // Open-Meteo does not require a key; reserved endpoints may use it later.
	client := &http.Client{Timeout: 12 * time.Second}
	location, err := fetchLocation(ctx, client, city)
	if err != nil {
		return Weather{}, Forecast{}, TodayOverview{}, err
	}
	query := url.Values{
		"latitude":      {strconv.FormatFloat(location.Latitude, 'f', 4, 64)},
		"longitude":     {strconv.FormatFloat(location.Longitude, 'f', 4, 64)},
		"current":       {"temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,wind_direction_10m,weather_code"},
		"daily":         {"weather_code,temperature_2m_max,temperature_2m_min,uv_index_max,sunrise,sunset"},
		"hourly":        {"precipitation_probability,precipitation"},
		"forecast_days": {"2"},
		"timezone":      {"auto"},
	}
	var forecast forecastResponse
	if err := getJSON(ctx, client, "https://api.open-meteo.com/v1/forecast?"+query.Encode(), &forecast); err != nil {
		return Weather{}, Forecast{}, TodayOverview{}, err
	}
	if len(forecast.Daily.WeatherCode) < 2 || len(forecast.Daily.TempMax) < 2 || len(forecast.Daily.TempMin) < 2 {
		return Weather{}, Forecast{}, TodayOverview{}, fmt.Errorf("weather forecast for %q is incomplete", city)
	}

	airQuality := "unknown"
	airQuery := url.Values{
		"latitude":  {strconv.FormatFloat(location.Latitude, 'f', 4, 64)},
		"longitude": {strconv.FormatFloat(location.Longitude, 'f', 4, 64)},
		"current":   {"us_aqi"},
		"timezone":  {"auto"},
	}
	var air airQualityResponse
	if err := getJSON(ctx, client, "https://air-quality-api.open-meteo.com/v1/air-quality?"+airQuery.Encode(), &air); err == nil {
		airQuality = airQualityLabel(air.Current.USAQI)
	}

	weatherCondition, weatherIcon := weatherDescription(forecast.Current.WeatherCode)
	tomorrowCondition, tomorrowIcon := weatherDescription(forecast.Daily.WeatherCode[1])
	uv := 0
	if len(forecast.Daily.UVMax) > 0 {
		uv = int(forecast.Daily.UVMax[0] + 0.5)
	}
	overview := TodayOverview{
		Sunrise: formatClock(forecast.Daily.Sunrise), Sunset: formatClock(forecast.Daily.Sunset),
		UVIndex: uv, UVLevel: uvLabel(uv), AirQuality: airQuality, UpdatedAt: time.Now(),
		RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, 0),
		RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, 0),
		TempHigh:        int(forecast.Daily.TempMax[0] + 0.5),
		TempLow:         int(forecast.Daily.TempMin[0] + 0.5),
	}
	if holiday, err := fetchNextHoliday(ctx, client, country, time.Now()); err == nil {
		overview.Holiday, overview.HolidayDate, overview.DaysUntil = holiday.Name, holiday.DateLabel, holiday.DaysUntil
	}
	return Weather{
			Condition: weatherCondition, Icon: weatherIcon,
			TempC: int(forecast.Current.Temperature + 0.5), FeelsLike: int(forecast.Current.FeelsLike + 0.5),
			Wind:            windArrow(forecast.Current.WindDirection) + " " + strconv.Itoa(int(forecast.Current.WindSpeed+0.5)) + " km/h",
			Humidity:        int(forecast.Current.Humidity + 0.5),
			RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, 0),
			RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, 0),
		}, Forecast{
			Condition: tomorrowCondition, Icon: tomorrowIcon,
			TempHigh: int(forecast.Daily.TempMax[1] + 0.5), TempLow: int(forecast.Daily.TempMin[1] + 0.5),
			RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, 24),
			RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, 24),
		}, overview, nil
}

func rainProbability(probabilities []int, dayOffset int) int {
	if dayOffset >= len(probabilities) {
		return 0
	}
	end := min(dayOffset+24, len(probabilities))
	maximum := 0
	for _, probability := range probabilities[dayOffset:end] {
		maximum = max(maximum, probability)
	}
	return maximum
}

func rainTime(times []string, probabilities []int, precipitation []float64, dayOffset int) string {
	if dayOffset >= len(probabilities) {
		return ""
	}
	end := min(dayOffset+24, len(probabilities))
	bestIndex, bestProbability := -1, 0
	for index := dayOffset; index < end; index++ {
		if probabilities[index] > bestProbability && (index >= len(precipitation) || precipitation[index] > 0) {
			bestIndex, bestProbability = index, probabilities[index]
		}
	}
	if bestIndex < 0 || bestIndex >= len(times) {
		return ""
	}
	if len(times[bestIndex]) >= 16 {
		return times[bestIndex][11:16]
	}
	return times[bestIndex]
}

type holidaySummary struct {
	Name      string
	DateLabel string
	DaysUntil int
}

func fetchNextHoliday(ctx context.Context, client *http.Client, country string, now time.Time) (holidaySummary, error) {
	country = strings.ToUpper(strings.TrimSpace(country))
	if len(country) != 2 {
		return holidaySummary{}, fmt.Errorf("invalid holiday country %q", country)
	}
	for yearOffset := 0; yearOffset <= 1; yearOffset++ {
		year := now.Year() + yearOffset
		var holidays []publicHoliday
		endpoint := fmt.Sprintf("https://date.nager.at/api/v3/PublicHolidays/%d/%s", year, url.PathEscape(country))
		if err := getJSON(ctx, client, endpoint, &holidays); err != nil {
			return holidaySummary{}, err
		}
		for _, holiday := range holidays {
			date, err := time.Parse("2006-01-02", holiday.Date)
			if err != nil || date.Before(now.Truncate(24*time.Hour)) {
				continue
			}
			name := holiday.LocalName
			if name == "" {
				name = holiday.Name
			}
			return holidaySummary{Name: name, DateLabel: date.Format("Jan 02"), DaysUntil: int(date.Sub(now).Hours() / 24)}, nil
		}
	}
	return holidaySummary{}, fmt.Errorf("no upcoming holidays found for %s", country)
}

func fetchLocation(ctx context.Context, client *http.Client, city string) (geoLocation, error) {
	query := url.Values{"name": {city}, "count": {"1"}, "language": {"en"}, "format": {"json"}}
	var response geocodingResponse
	if err := getJSON(ctx, client, "https://geocoding-api.open-meteo.com/v1/search?"+query.Encode(), &response); err != nil {
		return geoLocation{}, err
	}
	if len(response.Results) == 0 {
		return geoLocation{}, fmt.Errorf("city %q was not found", city)
	}
	return response.Results[0], nil
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("weather API: HTTP status %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 2<<20))
	return decoder.Decode(target)
}

func formatClock(values []string) string {
	if len(values) == 0 {
		return "--:--"
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04"} {
		parsed, err := time.Parse(layout, values[0])
		if err == nil {
			return parsed.Format("15:04")
		}
	}
	return "--:--"
}

func windArrow(degrees float64) string {
	arrows := []string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘"}
	index := int((degrees+22.5)/45) % len(arrows)
	return arrows[index]
}

func uvLabel(index int) string {
	switch {
	case index >= 11:
		return "extreme"
	case index >= 8:
		return "very high"
	case index >= 6:
		return "high"
	case index >= 3:
		return "moderate"
	default:
		return "low"
	}
}

func airQualityLabel(index float64) string {
	switch {
	case index <= 50:
		return "good"
	case index <= 100:
		return "moderate"
	case index <= 150:
		return "unhealthy for sensitive groups"
	case index > 150:
		return "unhealthy"
	default:
		return "unknown"
	}
}

func weatherDescription(code int) (string, []string) {
	switch {
	case code == 0:
		return "Clear sky", []string{`   \  /`, ` _ /\_.-. `, `   \_(   ).`, `   /(___(__)`}
	case code <= 3:
		return "Partly cloudy", []string{`   \  /`, ` _ /"".-. `, `   \_(   ).`, `   /(___(__)`}
	case code <= 48:
		return "Foggy", []string{" _ - _ - _ ", "  _ - _ -  ", " _ - _ - _ ", "            "}
	case code <= 67 || code >= 80 && code <= 82:
		return "Rain", []string{"     .-.   ", "    (   ).  ", "   (___(__) ", "    ' ' ' ' "}
	case code <= 77:
		return "Snow", []string{"     .-.   ", "    (   ).  ", "   * * * *  ", "    * * *   "}
	default:
		return "Thunderstorm", []string{"     .-.   ", "    (   ).  ", "   (___(__) ", "    ⚡ ⚡    "}
	}
}
