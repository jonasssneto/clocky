package data

// FetchAll aggregates the sources updated by the UI refresh cycle.
// The overview and markets remain loaded only during initialization.
func FetchAll() (Weather, Forecast, []NewsItem, []ContributionDay, Track) {
	return MockToday(), MockTomorrow(), MockNews(), MockContributions(), MockTrack()
}
