package data

import "time"

type TodayOverview struct {
	Sunrise     string
	Sunset      string
	UVIndex     int
	UVLevel     string
	AirQuality  string
	Holiday     string
	HolidayDate string
	DaysUntil   int
	UpdatedAt   time.Time
}

func MockTodayOverview() TodayOverview {
	return TodayOverview{
		Sunrise:     "06:03",
		Sunset:      "18:07",
		UVIndex:     7,
		UVLevel:     "high",
		AirQuality:  "good",
		Holiday:     "Our Lady of Aparecida",
		HolidayDate: "Oct 12",
		DaysUntil:   29,
		UpdatedAt:   time.Now(),
	}
}
