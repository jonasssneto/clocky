package data

// FetchAll aggregates the mock sources updated by the general UI refresh cycle.
// Spotify is refreshed independently because its provider performs network I/O.
// The overview and markets remain loaded only during initialization.
func FetchAll() (Weather, Forecast, []NewsItem, []ContributionDay) {
	return MockToday(), MockTomorrow(), MockNews(), MockContributions()
}
