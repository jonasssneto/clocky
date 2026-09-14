package data

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	marketRequestTimeout    = 12 * time.Second
	marketResponseBodyLimit = 2 << 20
	maximumMarketSymbolSize = 12
	marketHistoryCapacity   = 3
)

type capitalQuote struct {
	Ticker  string  `json:"ticker"`
	Price   float64 `json:"preco"`
	Range52 struct {
		Minimum float64 `json:"minima"`
		Maximum float64 `json:"maxima"`
	} `json:"faixa52"`
	Variations struct {
		Day float64 `json:"dia"`
	} `json:"variacoes"`
	Indicators struct {
		DividendYield12M *float64 `json:"dy_12m"`
	} `json:"indicadores"`
	MonthlyYield *float64 `json:"rendimento_mes_pct"`
}

// FetchMarketFunds gets public B3/FII quotes from Capital Agora.
func FetchMarketFunds(ctx context.Context, symbols []string) ([]MarketAsset, error) {
	clean := make([]string, 0, len(symbols))
	seen := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" || len(symbol) > maximumMarketSymbolSize {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		clean = append(clean, symbol)
	}
	if len(clean) == 0 {
		return nil, nil
	}
	client := &http.Client{Timeout: marketRequestTimeout}
	assets := make([]MarketAsset, 0, len(clean))
	for _, symbol := range clean {
		asset, ok := fetchMarketAsset(ctx, client, symbol)
		if ok {
			assets = append(assets, asset)
		}
	}
	if len(assets) == 0 {
		return nil, errors.New("market API returned no quotes for configured symbols")
	}
	return assets, nil
}

func fetchMarketAsset(ctx context.Context, client *http.Client, symbol string) (MarketAsset, bool) {
	endpoint := "https://capitalagora.com.br/api/ativo/" + symbol
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return MarketAsset{}, false
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Clocky market dashboard")
	response, err := client.Do(request)
	if err != nil {
		return MarketAsset{}, false
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return MarketAsset{}, false
	}
	var payload capitalQuote
	if err := json.NewDecoder(io.LimitReader(response.Body, marketResponseBodyLimit)).Decode(&payload); err != nil || payload.Ticker == "" || payload.Price == 0 {
		return MarketAsset{}, false
	}
	yield := "—"
	if payload.Indicators.DividendYield12M != nil {
		yield = formatYield(*payload.Indicators.DividendYield12M)
	} else if payload.MonthlyYield != nil {
		yield = formatYield(*payload.MonthlyYield) + "/m"
	}
	return MarketAsset{
		Symbol: payload.Ticker, Price: payload.Price, Change: payload.Variations.Day,
		DividendYield: yield, History: marketHistory(payload.Range52.Minimum, payload.Price, payload.Range52.Maximum),
	}, true
}

func marketHistory(minimum, current, maximum float64) []float64 {
	values := make([]float64, 0, marketHistoryCapacity)
	if minimum > 0 {
		values = append(values, minimum)
	}
	if current > 0 {
		values = append(values, current)
	}
	if maximum > 0 && maximum != current {
		values = append(values, maximum)
	}
	return values
}

func formatYield(value float64) string {
	if value == 0 {
		return "—"
	}
	return strconv.FormatFloat(value, 'f', 1, 64) + "%"
}
