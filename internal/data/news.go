package data

type NewsItem struct {
	Title   string
	Source  string
	Summary string
}

func MockNews() []NewsItem {
	return []NewsItem{
		{
			Title:   "Government announces new economic package",
			Source:  "G1",
			Summary: "The measures include new credit lines and tax changes for small businesses.",
		},
		{
			Title:   "Local team wins weekend derby",
			Source:  "GE",
			Summary: "The deciding goal came in the final minutes and changed the race for the top positions.",
		},
		{
			Title:   "Researchers discover species in the Amazon",
			Source:  "BBC News",
			Summary: "The new species was identified during an expedition in a rarely explored area.",
		},
		{
			Title:   "Market closes higher after inflation data",
			Source:  "InfoMoney",
			Summary: "Investors reacted to the latest indicators and expectations for upcoming interest rates.",
		},
		{
			Title:   "New study links sleep and productivity",
			Source:  "Folha",
			Summary: "The study followed professionals for six months and compared rest with performance.",
		},
	}
}
