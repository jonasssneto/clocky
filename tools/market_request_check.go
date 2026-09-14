// Command market_request_check checks the public market quote endpoint.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	symbols := os.Args[1:]
	if len(symbols) == 0 {
		symbols = []string{"MXRF11", "VINO11", "MTOF11", "VALE3", "PETR4"}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	for _, symbol := range symbols {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		endpoint := "https://capitalagora.com.br/api/ativo/" + symbol
		request, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			fmt.Printf("%s: request error: %v\n", symbol, err)
			continue
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("User-Agent", "Clocky market request check")
		response, err := client.Do(request)
		if err != nil {
			fmt.Printf("%s: network error: %v\n", symbol, err)
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		_ = response.Body.Close()
		fmt.Printf("%s: HTTP %d\n", symbol, response.StatusCode)
		if readErr != nil {
			fmt.Printf("  read error: %v\n", readErr)
			continue
		}
		var payload any
		if err := json.Unmarshal(body, &payload); err == nil {
			formatted, _ := json.MarshalIndent(payload, "  ", "  ")
			fmt.Printf("%s\n", formatted)
		} else {
			fmt.Printf("%s\n", body)
		}
	}
}
