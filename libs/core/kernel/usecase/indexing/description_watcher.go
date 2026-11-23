package indexing

// DescriptionWatcher reports when the description of a source changes.
type DescriptionWatcher interface {
	// WatchDescription registers fn to run each time the description changes.
	WatchDescription(fn func())
}
