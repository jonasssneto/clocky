package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"clocky/internal/data"
	"clocky/internal/imaging"
	"github.com/charmbracelet/x/ansi"
)

const (
	// Each "▀" combines two vertical pixels; 8x4 cells create an 8x8 cover.
	spotifyCoverWidth  = 8
	spotifyCoverHeight = 4
)

func formatTrackTime(seconds int) string {
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (m Model) spotifyStatus() string {
	if m.configErr != "" {
		return "Configuration server: " + m.configErr
	}
	return m.spotifyErr
}

func renderProgressBar(width, elapsed, total int) string {
	width = max(1, width)
	filled := 0
	if total > 0 {
		filled = min(width, max(0, width*elapsed/total))
	}
	return spotify.Render(strings.Repeat("━", filled)) + dim.Render(strings.Repeat("─", width-filled))
}

func renderPlaybackLine(artist string, width int, playing bool) string {
	icon := "▶"
	if playing {
		icon = "Ⅱ"
	}
	iconWidth := lipgloss.Width(icon)
	if width <= iconWidth {
		return spotify.Render(icon)
	}
	artist = ansi.Truncate(strings.TrimSpace(artist), width-iconWidth-1, "…")
	spacing := max(1, width-lipgloss.Width(artist)-iconWidth)
	return dim.Render(artist) + strings.Repeat(" ", spacing) + spotify.Render(icon)
}

func renderSpotifyBox(width, height int, track data.Track, cover, status, loginURL string) string {
	if height < boxStyle.GetVerticalFrameSize()+1 {
		return ""
	}
	innerWidth := max(1, width-boxStyle.GetHorizontalFrameSize())
	elapsed := formatTrackTime(track.ElapsedSec)
	duration := formatTrackTime(track.TotalSec)
	if cover == "" {
		cover = spotify.Render(imaging.RenderCover(nil, spotifyCoverWidth, spotifyCoverHeight))
	}
	if track.Title == "" {
		lines := []string{spotify.Render("Spotify"), truncateLine(status, innerWidth)}
		if loginURL != "" {
			lines = append(lines, dim.Render(truncateLine(loginURL, innerWidth)))
		}
		content := strings.Join(lines, "\n")
		if innerWidth >= spotifyCoverWidth+10 {
			content = lipgloss.JoinHorizontal(lipgloss.Top, cover, "  ", content)
		}
		return boxStyle.Width(width).Height(height).Render(content)
	}

	var content string
	if innerWidth >= spotifyCoverWidth+10 {
		infoWidth := innerWidth - lipgloss.Width(cover) - 2
		timeLabel := elapsed + " / " + duration
		if lipgloss.Width(timeLabel) > infoWidth {
			timeLabel = elapsed + "/" + duration
		}
		info := strings.Join([]string{
			clockStyle.MaxWidth(infoWidth).Render(track.Title),
			renderPlaybackLine(track.Artist, infoWidth, track.Playing),
			renderProgressBar(infoWidth, track.ElapsedSec, track.TotalSec),
			dim.Render(timeLabel),
		}, "\n")
		content = lipgloss.JoinHorizontal(lipgloss.Top, cover, "  ", info)
	} else {
		content = strings.Join([]string{
			clockStyle.MaxWidth(innerWidth).Render(track.Title),
			renderPlaybackLine(track.Artist, innerWidth, track.Playing),
			renderProgressBar(innerWidth, track.ElapsedSec, track.TotalSec),
			dim.Render(elapsed + "/" + duration),
		}, "\n")
	}
	return boxStyle.Width(width).Height(height).Render(content)
}
