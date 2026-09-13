package data

type MarketAsset struct {
	Symbol        string
	Price         float64
	Change        float64
	DividendYield string
	History       []float64
}

func MockStocks() []MarketAsset {
	return []MarketAsset{
		{Symbol: "VALE3", Price: 63.42, Change: 1.84, DividendYield: "8.6%", History: []float64{59.8, 60.4, 60.1, 61.2, 60.7, 61.9, 62.5, 62.1, 62.9, 63.4, 63.0, 63.42}},
		{Symbol: "ITSA3", Price: 11.18, Change: -0.35, DividendYield: "7.1%", History: []float64{11.55, 11.48, 11.52, 11.43, 11.39, 11.45, 11.34, 11.29, 11.32, 11.24, 11.22, 11.18}},
	}
}

func MockFunds() []MarketAsset {
	return []MarketAsset{
		{Symbol: "MXRF11", Price: 10.31, Change: 0.49, DividendYield: "12.1%", History: []float64{10.08, 10.12, 10.11, 10.18, 10.17, 10.24, 10.21, 10.27, 10.25, 10.29, 10.28, 10.31}},
		{Symbol: "VINO11", Price: 5.84, Change: -0.85, DividendYield: "14.8%", History: []float64{6.18, 6.12, 6.08, 6.11, 6.02, 5.98, 6.01, 5.94, 5.91, 5.87, 5.89, 5.84}},
	}
}
