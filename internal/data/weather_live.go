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

const (
	weatherRequestTimeout    = 12 * time.Second
	weatherResponseBodyLimit = 2 << 20
	coordinatePrecision      = 4
	floatBitSize             = 64
	forecastDayCount         = 2
	hoursPerForecastDay      = 24
	providerRoundingOffset   = 0.5
	holidaySearchYears       = 2
	nominalDayDuration       = 24 * time.Hour
	providerDateTimeLayout   = "2006-01-02T15:04"
	providerClockLayout      = "15:04"
	holidayDateLayout        = "2006-01-02"
	holidayLabelLayout       = "Jan 02"
	providerTimeStart        = 11
	providerTimeEnd          = 16
	windSectorOffset         = 22.5
	windSectorDegrees        = 45
	uvExtremeMinimum         = 11
	uvVeryHighMinimum        = 8
	uvHighMinimum            = 6
	uvModerateMinimum        = 3
	airQualityGoodMaximum    = 50
	airQualityModerateMax    = 100
	airQualitySensitiveMax   = 150
	clearSkyWeatherCode      = 0
	cloudyWeatherCodeMax     = 3
	fogWeatherCodeMax        = 48
	rainWeatherCodeMax       = 67
	snowWeatherCodeMax       = 77
	showerWeatherCodeMin     = 80
	showerWeatherCodeMax     = 82
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
	client := &http.Client{Timeout: weatherRequestTimeout}
	location, err := fetchLocation(ctx, client, city)
	if err != nil {
		return Weather{}, Forecast{}, TodayOverview{}, err
	}
	query := url.Values{
		"latitude":      {strconv.FormatFloat(location.Latitude, 'f', coordinatePrecision, floatBitSize)},
		"longitude":     {strconv.FormatFloat(location.Longitude, 'f', coordinatePrecision, floatBitSize)},
		"current":       {"temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,wind_direction_10m,weather_code"},
		"daily":         {"weather_code,temperature_2m_max,temperature_2m_min,uv_index_max,sunrise,sunset"},
		"hourly":        {"precipitation_probability,precipitation"},
		"forecast_days": {strconv.Itoa(forecastDayCount)},
		"timezone":      {"auto"},
	}
	var forecast forecastResponse
	if err := getJSON(ctx, client, "https://api.open-meteo.com/v1/forecast?"+query.Encode(), &forecast); err != nil {
		return Weather{}, Forecast{}, TodayOverview{}, err
	}
	if len(forecast.Daily.WeatherCode) < forecastDayCount || len(forecast.Daily.TempMax) < forecastDayCount || len(forecast.Daily.TempMin) < forecastDayCount {
		return Weather{}, Forecast{}, TodayOverview{}, fmt.Errorf("weather forecast for %q is incomplete", city)
	}

	airQuality := "unknown"
	airQuery := url.Values{
		"latitude":  {strconv.FormatFloat(location.Latitude, 'f', coordinatePrecision, floatBitSize)},
		"longitude": {strconv.FormatFloat(location.Longitude, 'f', coordinatePrecision, floatBitSize)},
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
		uv = roundedProviderValue(forecast.Daily.UVMax[0])
	}
	overview := TodayOverview{
		Sunrise: formatClock(forecast.Daily.Sunrise), Sunset: formatClock(forecast.Daily.Sunset),
		UVIndex: uv, UVLevel: uvLabel(uv), AirQuality: airQuality, UpdatedAt: time.Now(),
		RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, 0),
		RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, 0),
		TempHigh:        roundedProviderValue(forecast.Daily.TempMax[0]),
		TempLow:         roundedProviderValue(forecast.Daily.TempMin[0]),
	}
	if holiday, err := fetchNextHoliday(ctx, client, country, time.Now()); err == nil {
		overview.Holiday, overview.HolidayDate, overview.DaysUntil = holiday.Name, holiday.DateLabel, holiday.DaysUntil
	}
	today := Weather{
		Condition: weatherCondition, Icon: weatherIcon,
		TempC: roundedProviderValue(forecast.Current.Temperature), FeelsLike: roundedProviderValue(forecast.Current.FeelsLike),
		Wind:            windArrow(forecast.Current.WindDirection) + " " + strconv.Itoa(roundedProviderValue(forecast.Current.WindSpeed)) + " km/h",
		Humidity:        roundedProviderValue(forecast.Current.Humidity),
		RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, 0),
		RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, 0),
	}
	tomorrow := Forecast{
		Condition: tomorrowCondition, Icon: tomorrowIcon,
		TempHigh: roundedProviderValue(forecast.Daily.TempMax[1]), TempLow: roundedProviderValue(forecast.Daily.TempMin[1]),
		RainProbability: rainProbability(forecast.Hourly.PrecipitationProbability, hoursPerForecastDay),
		RainTime:        rainTime(forecast.Hourly.Time, forecast.Hourly.PrecipitationProbability, forecast.Hourly.Precipitation, hoursPerForecastDay),
	}
	return today, tomorrow, overview, nil
}

func roundedProviderValue(value float64) int {
	return int(value + providerRoundingOffset)
}

func rainProbability(probabilities []int, dayOffset int) int {
	if dayOffset >= len(probabilities) {
		return 0
	}
	end := min(dayOffset+hoursPerForecastDay, len(probabilities))
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
	end := min(dayOffset+hoursPerForecastDay, len(probabilities))
	bestIndex, bestProbability := -1, 0
	for index := dayOffset; index < end; index++ {
		if probabilities[index] > bestProbability && (index >= len(precipitation) || precipitation[index] > 0) {
			bestIndex, bestProbability = index, probabilities[index]
		}
	}
	if bestIndex < 0 {
		return ""
	}
	if bestIndex >= len(times) {
		return ""
	}
	selectedTime := times[bestIndex]
	if len(selectedTime) >= providerTimeEnd {
		return selectedTime[providerTimeStart:providerTimeEnd]
	}
	return selectedTime
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
	for yearOffset := range holidaySearchYears {
		year := now.Year() + yearOffset
		var holidays []publicHoliday
		endpoint := fmt.Sprintf("https://date.nager.at/api/v3/PublicHolidays/%d/%s", year, url.PathEscape(country))
		if err := getJSON(ctx, client, endpoint, &holidays); err != nil {
			return holidaySummary{}, err
		}
		for _, holiday := range holidays {
			date, err := time.Parse(holidayDateLayout, holiday.Date)
			if err != nil || date.Before(now.Truncate(nominalDayDuration)) {
				continue
			}
			name := holiday.LocalName
			if name == "" {
				name = holiday.Name
			}
			return holidaySummary{Name: name, DateLabel: date.Format(holidayLabelLayout), DaysUntil: int(date.Sub(now).Hours() / hoursPerForecastDay)}, nil
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
	decoder := json.NewDecoder(io.LimitReader(response.Body, weatherResponseBodyLimit))
	return decoder.Decode(target)
}

func formatClock(values []string) string {
	if len(values) == 0 {
		return "--:--"
	}
	parsed, err := time.Parse(time.RFC3339, values[0])
	if err != nil {
		parsed, err = time.Parse(providerDateTimeLayout, values[0])
	}
	if err == nil {
		return parsed.Format(providerClockLayout)
	}
	return "--:--"
}

func windArrow(degrees float64) string {
	arrows := []string{"↓", "↙", "←", "↖", "↑", "↗", "→", "↘"}
	index := int((degrees+windSectorOffset)/windSectorDegrees) % len(arrows)
	return arrows[index]
}

func uvLabel(index int) string {
	switch {
	case index >= uvExtremeMinimum:
		return "extreme"
	case index >= uvVeryHighMinimum:
		return "very high"
	case index >= uvHighMinimum:
		return "high"
	case index >= uvModerateMinimum:
		return "moderate"
	default:
		return "low"
	}
}

func airQualityLabel(index float64) string {
	switch {
	case index <= airQualityGoodMaximum:
		return "good"
	case index <= airQualityModerateMax:
		return "moderate"
	case index <= airQualitySensitiveMax:
		return "unhealthy for sensitive groups"
	case index > airQualitySensitiveMax:
		return "unhealthy"
	default:
		return "unknown"
	}
}

func weatherDescription(code int) (string, []string) {
	switch {
	case code == clearSkyWeatherCode:
		return "Clear sky", []string{`   \  /`, ` _ /\_.-. `, `   \_(   ).`, `   /(___(__)`}
	case code <= cloudyWeatherCodeMax:
		return "Partly cloudy", []string{`   \  /`, ` _ /"".-. `, `   \_(   ).`, `   /(___(__)`}
	case code <= fogWeatherCodeMax:
		return "Foggy", []string{" _ - _ - _ ", "  _ - _ -  ", " _ - _ - _ ", "            "}
	case code <= rainWeatherCodeMax || code >= showerWeatherCodeMin && code <= showerWeatherCodeMax:
		return "Rain", []string{"     .-.   ", "    (   ).  ", "   (___(__) ", "    ' ' ' ' "}
	case code <= snowWeatherCodeMax:
		return "Snow", []string{"     .-.   ", "    (   ).  ", "   * * * *  ", "    * * *   "}
	default:
		return "Thunderstorm", []string{"     .-.   ", "    (   ).  ", "   (___(__) ", "    ⚡ ⚡    "}
	}
}
