package ui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	minimumDashboardDimension = 1
	minimumBottomColumnsWidth = 30
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

func renderTodayNewsBox(width, height int, m Model) string {
	if m.todayNewsPage {
		return renderNewsBox(width, height, m.lastRefresh, m.news, m.newsLimit)
	}
	return renderTodayBox(width, height, m.now, m.overview)
}

func renderDashboardBottom(m Model, leftWidth, rightWidth, height, mediaHeight int) string {
	switchHeight := height - mediaHeight
	leftColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		renderMarketsBox(leftWidth, height-mediaHeight, m.marketPage, m.marketNext, m.marketStep, m.marketSlide, m.stocks, m.funds),
		renderSpotifyBox(leftWidth, mediaHeight, m.track, m.cover, m.spotifyStatus(), m.spotifyPage),
	)
	rightContent := lipgloss.JoinVertical(
		lipgloss.Left,
		renderTodayNewsBox(rightWidth, switchHeight, m),
		renderGithubBox(rightWidth, mediaHeight, m.lastRefresh, m.github, m.githubTotal),
	)
	rightColumn := lipgloss.NewStyle().Height(height).AlignVertical(lipgloss.Center).Render(rightContent)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, " ", rightColumn)
}

func (m Model) View() tea.View {
	if m.settings != nil {
		applyVisualSettings(m.settings.Get())
	}
	width := max(minimumDashboardDimension, m.width)
	height := max(minimumDashboardDimension, m.height)
	if m.update != nil {
		view := tea.NewView(renderUpdateProgress(width, height, *m.update))
		view.AltScreen = true
		return view
	}
	columnGap := visualColumnGap

	todayCol := renderTodayColumn(m.today, weatherTitle(m.weatherCity, "Today"))
	tomorrowCol := renderTomorrowColumn(m.tomorrow, weatherTitle(m.weatherCity, "Tomorrow"))
	frameWidth := boxStyle.GetHorizontalFrameSize()
	compactClockWidth := max(
		lipgloss.Width(m.now.Format(clockTimeLayout)),
		lipgloss.Width(m.now.Format(clockDateLayout)),
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
		weatherContent := renderWeather(m.today, m.tomorrow, rightWidth, m.weatherCity, m.weatherTomorrow)
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
		weatherContent := renderWeather(m.today, m.tomorrow, width, m.weatherCity, m.weatherTomorrow)
		clockBox := boxStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(clockContent)
		weatherBox := boxStyle.Width(width).Render(weatherContent)
		top = lipgloss.JoinVertical(lipgloss.Left, clockBox, weatherBox)
	}

	full := top
	bottomHeight := height - lipgloss.Height(top)
	baseMediaHeight := boxStyle.GetVerticalFrameSize() + spotifyCoverHeight
	if !twoColumns && width >= minimumBottomColumnsWidth {
		leftWidth, rightWidth = splitColumns(width, columnGap)
	}
	if (twoColumns || width >= minimumBottomColumnsWidth) && bottomHeight >= baseMediaHeight*2 {
		bottom := renderDashboardBottom(m, leftWidth, rightWidth, bottomHeight, baseMediaHeight)
		full = lipgloss.JoinVertical(lipgloss.Left, top, bottom)
	}

	view := tea.NewView(full)
	view.AltScreen = true
	return view
}
