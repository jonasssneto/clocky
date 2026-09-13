# Clocky Agent Guide

## Project purpose

Clocky is a responsive terminal dashboard written in Go. It uses Bubble Tea v2 for the application runtime, Lip Gloss v2 for layout and styling, and ANSI-aware helpers for terminal-safe truncation and rendering.

The current phase is UI-first. Dashboard values are intentionally mocked so the layout and behavior can be developed before real data providers are connected.

## Core rules

- Write all source code, identifiers, comments, errors, build messages, mock content, and visible UI text in English.
- Use Bubble Tea v2 APIs and documentation only. Do not copy examples written for Bubble Tea v1 without verifying the v2 equivalent.
- Keep all dashboard data mocked unless the user explicitly requests a real integration.
- Preserve responsive behavior. Never design a card for only one terminal size.
- Keep `main.go` free of application logic. It may only create `tea.NewProgram(ui.NewModel())`, run it, and handle the returned error.
- Keep domain data independent from Bubble Tea and Lip Gloss.
- Do not perform blocking network or disk work inside `Update()` or `View()`. Use `tea.Cmd` for asynchronous work.
- Do not commit generated binaries from `bin/`.

## Architecture

```text
clocky/
├── go.mod
├── main.go
└── internal/
    ├── data/
    │   ├── weather.go
    │   ├── news.go
    │   ├── github.go
    │   ├── spotify.go
    │   ├── today.go
    │   ├── markets.go
    │   └── fetch.go
    ├── imaging/
    │   └── cover.go
    └── ui/
        ├── styles.go
        ├── messages.go
        ├── model.go
        ├── update.go
        ├── view.go
        ├── clock.go
        ├── weather.go
        ├── today.go
        ├── markets.go
        ├── news.go
        ├── github.go
        ├── spotify.go
        └── helpers.go
```

### `main.go`

This is only the application entrypoint. Do not move state, data loading, commands, rendering, or layout logic into it.

### `internal/data`

This package owns domain types and data sources. Each file owns one domain:

- `weather.go`: `Weather`, `Forecast`, `MockToday()`, and `MockTomorrow()`.
- `news.go`: `NewsItem` and `MockNews()`.
- `github.go`: `ContributionDay` and `MockContributions()`.
- `spotify.go`: `Track` and `MockTrack()`.
- `today.go`: `TodayOverview` and `MockTodayOverview()`.
- `markets.go`: `MarketAsset`, `MockStocks()`, and `MockFunds()`.
- `fetch.go`: `FetchAll()`, which aggregates data refreshed by the UI.

The UI must consume exported domain values such as `data.Weather` and `data.Track`; it must not know whether those values came from a mock or a real API.

When real providers are added, replace or extend the implementation inside the relevant domain file. Do not put API-specific parsing, authentication, HTTP payloads, or provider response types in `internal/ui`.

`FetchAll()` currently refreshes weather, tomorrow's forecast, news, GitHub contributions, and the Spotify track. The today overview, stocks, and funds are loaded once by `ui.NewModel()`. Preserve this distinction unless refresh behavior is intentionally changed.

### `internal/imaging`

This package owns Spotify cover infrastructure:

- Download and decode the JPEG with a bounded response body and an HTTP timeout.
- Convert an `image.Image` into ANSI true-color half-blocks.
- Render the placeholder used before a cover is available.

Keep this package independent from Bubble Tea model state. `RenderCover()` must remain usable without creating a Tea program.

### `internal/ui`

This package owns all Bubble Tea and Lip Gloss behavior:

- `styles.go`: shared colors, borders, and styles.
- `messages.go`: message types, interval constants, and command constructors.
- `model.go`: `Model`, `NewModel()`, and `Init()`.
- `update.go`: `Model.Update()` only.
- `view.go`: `Model.View()` and top-level layout composition.
- One card file per dashboard domain.
- `helpers.go`: only generic, cross-card rendering helpers.

Do not import `net/http`, provider SDKs, or API response models into this package.

## Current UI behavior

Preserve these behaviors unless a request explicitly changes them:

- The clock and weather cards share the same height when displayed on the same row.
- The large clock falls back to a compact clock when its ASCII font does not fit.
- Wide weather cards show Today and Tomorrow side by side.
- Compact weather cards remove Tomorrow and preserve Today's icon, temperature, feels-like temperature, wind, and humidity.
- News shows only the top three titles and sources.
- GitHub shows a seven-day contribution heatmap plus yearly and weekly totals.
- The Today card shows daylight, UV index, air quality, and the next holiday when height permits.
- Markets keeps Funds on the left and Stocks on the right with a vertical separator.
- Funds and Stocks use synchronized, dedicated slides for each asset.
- Market trends use responsive Braille line charts and color the line green for a positive change or red for a negative change.
- Each market asset shows its symbol, price, and dividend yield.
- Spotify shows a cover, title, artist, progress, elapsed time, duration, and a right-aligned play/pause status.
- The Spotify cover uses 10 columns by 5 terminal rows. Because each `▀` combines two vertical pixels, this represents a visually square 10-by-10 image.
- `ctrl+c` is the application exit command.

## Responsive layout rules

- Read available dimensions from `tea.WindowSizeMsg` and store them on the model.
- Use `lipgloss.Width()` and `lipgloss.Height()` for rendered content. Do not use byte length to measure terminal width.
- Use ANSI-aware truncation for styled strings. Text that exceeds its available width must end with `…`.
- Include borders and padding when calculating card dimensions through `boxStyle.GetHorizontalFrameSize()` and `boxStyle.GetVerticalFrameSize()`.
- Allocate widths from the current terminal width. Do not hard-code a complete dashboard width.
- Keep horizontal siblings within the exact available width, including separators and gaps.
- When content cannot fit, prefer a simpler dedicated compact view instead of squeezing every field into the card.
- Ensure animation frames have the same rendered width and height as their static states.

## Bubble Tea behavior

- The project targets `charm.land/bubbletea/v2`.
- Use `tea.NewView()` from Bubble Tea v2 and keep alternate-screen mode enabled.
- Timers and animation frames must return their next `tea.Cmd`; never sleep inside the update loop.
- Cover downloads must return a message and be accepted only when the response URL still matches the current track.
- Keep market slide state transitions in `Update()` and rendering in the market card functions.
- Keep `View()` deterministic and free of I/O.

## Data and mock conventions

- Mock values should be realistic enough to exercise the UI.
- Mock collections must return new slices instead of exposing shared mutable global state.
- Keep units explicit in rendered labels.
- `R$` and Brazilian market tickers are domain data and should not be translated.
- Use English field names such as `DividendYield`; avoid local abbreviations in identifiers.
- Preserve enough history points for the market renderer to interpolate across different card widths.

## Imaging conventions

- Keep the HTTP timeout and response-size limit when modifying cover downloads.
- Validate successful HTTP status codes before decoding.
- Keep ANSI color generation inside `internal/imaging`.
- Account for the two vertical image samples represented by each half-block character.
- If cover dimensions change, update the responsive Spotify width threshold and verify the full dashboard layout.

## Validation workflow

Run these commands after code changes:

```sh
gofmt -w main.go internal/data internal/imaging internal/ui
go test ./...
go vet ./...
git diff --check
```

For build-related changes, also run:

```sh
make build
make build-arm64
```

The deployment machine is Linux ARM64 (`aarch64`), so its executable must be built with `GOOS=linux`, `GOARCH=arm64`, and `CGO_ENABLED=0`. Do not send the native x86-64 binary to that machine.

## Git conventions

- Keep commits small and separated by responsibility.
- Use concise English commit messages, preferably Conventional Commit style.
- Do not rewrite, squash, or amend existing commits unless explicitly requested.
- Do not include generated binaries or timestamps or unrelated local files in commits.
