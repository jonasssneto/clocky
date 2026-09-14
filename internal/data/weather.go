package data

// Weather represents the current conditions.
type Weather struct {
	Condition       string
	Icon            []string
	TempC           int
	FeelsLike       int
	Wind            string
	Humidity        int
	RainProbability int
	RainTime        string
}

// Forecast represents the summarized conditions for a future day.
type Forecast struct {
	Condition       string
	Icon            []string
	TempHigh        int
	TempLow         int
	RainProbability int
	RainTime        string
}

func MockToday() Weather {
	return Weather{
		Condition: "Partly cloudy",
		Icon: []string{
			`   \  /`,
			` _ /"".-.    `,
			`   \_(   ).  `,
			`   /(___(__) `,
		},
		TempC:     24,
		FeelsLike: 26,
		Wind:      "↘ 11 km/h",
		Humidity:  68,
	}
}

func MockTomorrow() Forecast {
	return Forecast{
		Condition: "Light rain",
		Icon: []string{
			`     .-.   `,
			`    (   ).  `,
			`   (___(__) `,
			`    ' ' ' ' `,
		},
		TempHigh: 22,
		TempLow:  16,
	}
}
