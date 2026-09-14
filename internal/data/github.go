package data

import "fmt"

type ContributionDay struct {
	Label string
	Count int
	Level int
}

func MockContributions() []ContributionDay {
	days := make([]ContributionDay, 0, 30)
	for day := 0; day < 30; day++ {
		count := (day * 3) % 6
		days = append(days, ContributionDay{
			Label: fmt.Sprintf("%02d", day+1), Count: count, Level: min(4, count),
		})
	}
	return days
}
