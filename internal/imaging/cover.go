package imaging

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	coverRequestTimeout    = 10 * time.Second
	coverResponseBodyLimit = 5 << 20
	minimumCoverDimension  = 3
)

func DownloadAndRenderCover(url string, width, height int) (string, error) {
	client := http.Client{Timeout: coverRequestTimeout}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("cover: HTTP status %d", response.StatusCode)
	}
	cover, err := jpeg.Decode(io.LimitReader(response.Body, coverResponseBodyLimit))
	if err != nil {
		return "", err
	}
	return RenderCover(cover, width, height), nil
}

func RenderCover(cover image.Image, width, height int) string {
	width = max(minimumCoverDimension, width)
	height = max(minimumCoverDimension, height)
	if cover == nil {
		innerWidth := width - 2
		lines := []string{"┌" + strings.Repeat("─", innerWidth) + "┐"}
		for row := 0; row < height-2; row++ {
			content := ""
			if row == (height-2)/2 {
				content = "♪"
			}
			contentWidth := 0
			if content != "" {
				contentWidth = 1
			}
			leftPadding := max(0, (innerWidth-contentWidth)/2)
			rightPadding := max(0, innerWidth-leftPadding-contentWidth)
			lines = append(lines, "│"+strings.Repeat(" ", leftPadding)+content+strings.Repeat(" ", rightPadding)+"│")
		}
		lines = append(lines, "└"+strings.Repeat("─", innerWidth)+"┘")
		return strings.Join(lines, "\n")
	}

	bounds := cover.Bounds()
	var rendered strings.Builder
	for row := 0; row < height; row++ {
		if row > 0 {
			rendered.WriteByte('\n')
		}
		for column := 0; column < width; column++ {
			x := bounds.Min.X + (2*column+1)*bounds.Dx()/(2*width)
			topY := bounds.Min.Y + (2*row+1)*bounds.Dy()/(2*height)
			bottomY := bounds.Min.Y + (2*row+2)*bounds.Dy()/(2*height)
			bottomY = min(bottomY, bounds.Max.Y-1)
			topR, topG, topB, _ := cover.At(x, topY).RGBA()
			bottomR, bottomG, bottomB, _ := cover.At(x, bottomY).RGBA()
			fmt.Fprintf(
				&rendered,
				"\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
				topR>>8, topG>>8, topB>>8,
				bottomR>>8, bottomG>>8, bottomB>>8,
			)
		}
	}
	return rendered.String()
}
