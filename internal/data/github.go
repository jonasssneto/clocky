package data

type ContributionDay struct {
	Label string
	Count int
	Level int
}

func MockContributions() []ContributionDay {
	return []ContributionDay{
		{Label: "Mon", Count: 0, Level: 0},
		{Label: "Tue", Count: 1, Level: 1},
		{Label: "Wed", Count: 0, Level: 0},
		{Label: "Thu", Count: 2, Level: 2},
		{Label: "Fri", Count: 3, Level: 3},
		{Label: "Sat", Count: 1, Level: 1},
		{Label: "Sun", Count: 3, Level: 3},
	}
}
