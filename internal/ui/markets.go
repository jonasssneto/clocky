package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
	"github.com/charmbracelet/x/ansi"
)

func formatBrazilianReal(price float64) string {
	formatted := fmt.Sprintf("%.2f", price)
	return "R$ " + strings.Replace(formatted, ".", ",", 1)
}

func fitLine(text string, width int) string {
	line := ansi.Truncate(text, width, "…")
	return line + strings.Repeat(" ", max(0, width-lipgloss.Width(line)))
}

func renderLineChart(values []float64, width, height int, rising bool) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	if len(values) == 0 {
		return make([]string, height)
	}

	minimum, maximum := values[0], values[0]
	for _, point := range values[1:] {
		minimum = min(minimum, point)
		maximum = max(maximum, point)
	}
	span := maximum - minimum
	if span == 0 {
		span = 1
	}

	canvasWidth, canvasHeight := width*2, height*4
	points := make([]int, canvasWidth)
	for x := range points {
		if canvasWidth == 1 || len(values) == 1 {
			points[x] = canvasHeight / 2
			continue
		}
		position := float64(x) * float64(len(values)-1) / float64(canvasWidth-1)
		left := int(position)
		right := min(left+1, len(values)-1)
		fraction := position - float64(left)
		point := values[left] + (values[right]-values[left])*fraction
		points[x] = int((maximum - point) / span * float64(canvasHeight-1))
	}

	canvas := make([][]uint8, height)
	for row := range canvas {
		canvas[row] = make([]uint8, width)
	}
	setDot := func(x, y int) {
		if x < 0 || x >= canvasWidth || y < 0 || y >= canvasHeight {
			return
		}
		bits := [4][2]uint8{{1, 8}, {2, 16}, {4, 32}, {64, 128}}
		canvas[y/4][x/2] |= bits[y%4][x%2]
	}
	for x := 0; x < canvasWidth-1; x++ {
		y0, y1 := points[x], points[x+1]
		step := 1
		if y1 < y0 {
			step = -1
		}
		setDot(x, y0)
		for y := y0; y != y1; y += step {
			setDot(x, y)
			setDot(x+1, y+step)
		}
	}
	setDot(canvasWidth-1, points[canvasWidth-1])

	style := negative
	if rising {
		style = positive
	}
	chart := make([]string, height)
	for row := range canvas {
		var line strings.Builder
		for _, dots := range canvas[row] {
			if dots == 0 {
				line.WriteByte(' ')
				continue
			}
			line.WriteRune(rune(0x2800) + rune(dots))
		}
		chart[row] = style.Render(line.String())
	}
	return chart
}

func renderMarketColumn(width, height int, title string, selected bool, assets []data.MarketAsset, page int) []string {
	if height <= 0 {
		return nil
	}
	if len(assets) == 0 {
		return []string{fitLine(title+" · no data", width)}
	}
	page = max(0, min(page, len(assets)-1))
	asset := assets[page]
	headerTitle := dim.Render(title)
	if selected {
		headerTitle = value.Render(title)
	}
	position := page + 1
	indicator := strings.Repeat("○ ", max(0, position-1)) + "●" + strings.Repeat(" ○", max(0, len(assets)-position))
	header := headerTitle + fmt.Sprintf(" %d/%d ", position, len(assets)) + dim.Render(indicator)
	if height == 1 {
		return []string{fitLine(fmt.Sprintf("%s · %s", title, asset.Symbol), width)}
	}

	lines := []string{header}
	compactPrice := strings.TrimPrefix(formatBrazilianReal(asset.Price), "R$ ")
	assetLine := fmt.Sprintf("%s %s Yld %s", asset.Symbol, compactPrice, asset.DividendYield)
	fullAssetLine := fmt.Sprintf("%s %s · Yield %s", asset.Symbol, formatBrazilianReal(asset.Price), asset.DividendYield)
	if lipgloss.Width(fullAssetLine) <= width {
		assetLine = fullAssetLine
	}
	lines = append(lines, clockStyle.Render(assetLine))

	chartHeight := min(3, height-len(lines))
	if chartHeight > 0 {
		lines = append(lines, renderLineChart(asset.History, width, chartHeight, asset.Change >= 0)...)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for index, line := range lines {
		lines[index] = fitLine(line, width)
	}
	return lines
}

func renderMarketPanel(width, height, page int, stocks, funds []data.MarketAsset) []string {
	if height <= 0 {
		return nil
	}
	if len(stocks) == 0 && len(funds) == 0 {
		return []string{fitLine("Markets · no data", width)}
	}
	page %= 2
	leftWidth := max(1, width/2)
	rightWidth := max(1, width-1-leftWidth)
	leftLines := renderMarketColumn(leftWidth, height, "Funds", page == 0, funds, page)
	rightLines := renderMarketColumn(rightWidth, height, "Stocks", page == 1, stocks, page)
	lines := make([]string, height)
	for index := range lines {
		lines[index] = fitLine(leftLines[index], leftWidth) + dim.Render("│") + fitLine(rightLines[index], rightWidth)
	}
	return lines
}

func renderMarketsBox(width, height, page, next, step int, sliding bool, stocks, funds []data.MarketAsset) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	innerHeight := height - boxStyle.GetVerticalFrameSize()
	currentLines := renderMarketPanel(innerWidth, innerHeight, page, stocks, funds)
	if !sliding {
		return boxStyle.Width(width).Height(height).Render(strings.Join(currentLines, "\n"))
	}

	nextLines := renderMarketPanel(innerWidth, innerHeight, next, stocks, funds)
	offset := (innerWidth + 1) * step / marketSlideSteps
	visible := make([]string, innerHeight)
	for index := range visible {
		track := currentLines[index] + " " + nextLines[index]
		visible[index] = fitLine(ansi.Cut(track, offset, offset+innerWidth), innerWidth)
	}
	return boxStyle.Width(width).Height(height).Render(strings.Join(visible, "\n"))
}
