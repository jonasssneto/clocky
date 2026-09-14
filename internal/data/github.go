package data

import "fmt"

type ContributionDay struct {
	Label string
	Count int
	Level int
}

func MockContributions() []ContributionDay {
	days := make([]ContributionDay, 0, 30*7)
	for column := 0; column < 30; column++ {
		for row := 0; row < 7; row++ {
			count := (column*3 + row*5) % 6
			days = append(days, ContributionDay{
				Label: fmt.Sprintf("%02d", column+1), Count: count, Level: min(4, count),
			})
		}
	}
	return days
}
