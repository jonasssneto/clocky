package data

import "time"

type TodayOverview struct {
	Sunrise         string
	Sunset          string
	UVIndex         int
	UVLevel         string
	AirQuality      string
	RainProbability int
	RainTime        string
	TempHigh        int
	TempLow         int
	Holiday         string
	HolidayDate     string
	DaysUntil       int
	UpdatedAt       time.Time
}

func MockTodayOverview() TodayOverview {
	return TodayOverview{
		Sunrise:     "06:03",
		Sunset:      "18:07",
		UVIndex:     7,
		UVLevel:     "high",
		AirQuality:  "good",
		TempHigh:    28,
		TempLow:     20,
		Holiday:     "Our Lady of Aparecida",
		HolidayDate: "Oct 12",
		DaysUntil:   29,
		UpdatedAt:   time.Now(),
	}
}
