package data

// FetchAll aggregates the currently configured non-Spotify dashboard sources.
// Weather is fetched asynchronously by the UI so network work never blocks Update.
func FetchAll() (Weather, Forecast, []NewsItem, []ContributionDay) {
	return MockToday(), MockTomorrow(), MockNews(), MockContributions()
}
