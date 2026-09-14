package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func renderTodayMarketsRow(width, height int, m Model) string {
	compactHeight := boxStyle.GetVerticalFrameSize() + 1
	if height >= compactHeight*2 {
		todayHeight := min(boxStyle.GetVerticalFrameSize()+4*visualScale/100, height-compactHeight)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			renderTodayBox(width, todayHeight, m.now, m.overview),
			renderMarketsBox(width, height-todayHeight, m.marketPage, m.marketNext, m.marketStep, m.marketSlide, m.stocks, m.funds),
		)
	}

	const gap = 1
	todayWidth, marketsWidth := splitColumns(width, gap)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderTodayBox(todayWidth, height, m.now, m.overview),
		" ",
		renderMarketsBox(marketsWidth, height, m.marketPage, m.marketNext, m.marketStep, m.marketSlide, m.stocks, m.funds),
	)
}

func splitColumns(width, gap int) (int, int) {
	availableWidth := max(2, width-gap)
	leftWidth := (availableWidth + 1) / 2
	return leftWidth, availableWidth - leftWidth
}

func (m Model) View() tea.View {
	if m.settings != nil {
		applyVisualSettings(m.settings.Get())
	}
	width, height := max(1, m.width), max(1, m.height)
	columnGap := visualColumnGap

	todayCol := renderTodayColumn(m.today)
	tomorrowCol := renderTomorrowColumn(m.tomorrow)
	frameWidth := boxStyle.GetHorizontalFrameSize()
	compactClockWidth := max(
		lipgloss.Width(m.now.Format("15:04:05")),
		lipgloss.Width(m.now.Format("Mon, 02 Jan 2006")),
	) + frameWidth
	stackedWeatherWidth := max(lipgloss.Width(todayCol), lipgloss.Width(tomorrowCol)) + frameWidth

	twoColumns := width >= compactClockWidth+columnGap+stackedWeatherWidth
	leftWidth, rightWidth := width, width
	if twoColumns {
		leftWidth, rightWidth = splitColumns(width, columnGap)
	}

	var top string
	if twoColumns {
		clockContent := renderClock(m.now, leftWidth)
		weatherContent := renderWeather(m.today, m.tomorrow, rightWidth)
		clockCardStyle := boxStyle.Width(leftWidth).Align(lipgloss.Center, lipgloss.Center)
		weatherCardStyle := boxStyle.Width(rightWidth)
		clockBox := clockCardStyle.Render(clockContent)
		weatherBox := weatherCardStyle.Render(weatherContent)
		topCardHeight := max(lipgloss.Height(clockBox), lipgloss.Height(weatherBox))
		clockBox = clockCardStyle.Height(topCardHeight).Render(clockContent)
		weatherBox = weatherCardStyle.Height(topCardHeight).Render(weatherContent)
		top = lipgloss.JoinHorizontal(lipgloss.Top, clockBox, " ", weatherBox)
	} else {
		clockContent := renderClock(m.now, compactClockWidth)
		weatherContent := renderCompactWeather(m.today, width)
		clockBox := boxStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(clockContent)
		weatherBox := boxStyle.Width(width).Render(weatherContent)
		top = lipgloss.JoinVertical(lipgloss.Left, clockBox, weatherBox)
	}

	full := top
	bottomHeight := height - lipgloss.Height(top)
	mediaHeight := boxStyle.GetVerticalFrameSize() + spotifyCoverHeight
	if twoColumns && bottomHeight >= boxStyle.GetVerticalFrameSize()+1+mediaHeight {
		newsHeight := bottomHeight - mediaHeight
		leftColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			renderTodayMarketsRow(leftWidth, newsHeight, m),
			renderSpotifyBox(leftWidth, mediaHeight, m.track, m.cover, m.spotifyStatus(), m.spotifyPage),
		)
		rightColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			renderNewsBox(rightWidth, newsHeight, m.lastRefresh, m.news),
			renderGithubBox(rightWidth, mediaHeight, m.lastRefresh, m.github),
		)
		bottom := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, " ", rightColumn)
		full = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	} else if !twoColumns && width >= 30 && bottomHeight >= boxStyle.GetVerticalFrameSize()+1+mediaHeight {
		leftWidth, rightWidth = splitColumns(width, columnGap)
		newsHeight := bottomHeight - mediaHeight
		leftColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			renderTodayMarketsRow(leftWidth, newsHeight, m),
			renderSpotifyBox(leftWidth, mediaHeight, m.track, m.cover, m.spotifyStatus(), m.spotifyPage),
		)
		rightColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			renderNewsBox(rightWidth, newsHeight, m.lastRefresh, m.news),
			renderGithubBox(rightWidth, mediaHeight, m.lastRefresh, m.github),
		)
		bottom := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, " ", rightColumn)
		full = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	}

	view := tea.NewView(full)
	view.AltScreen = true
	return view
}
