package data

type Track struct {
	Title      string
	Artist     string
	Playing    bool
	ElapsedSec int
	TotalSec   int
	CoverURL   string
}

func MockTrack() Track {
	return Track{
		Title:      "Midnight City",
		Artist:     "M83",
		Playing:    true,
		ElapsedSec: 102,
		TotalSec:   243,
		CoverURL:   "https://i.scdn.co/image/ab67616d0000b273e3e3b64cea45265469d4cafa",
	}
}
