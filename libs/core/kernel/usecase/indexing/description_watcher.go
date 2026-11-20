package indexing

// DescriptionWatcher reports when the description of a source changes.
type DescriptionWatcher interface {
	// OnDescriptionChange registers fn to run each time the description changes.
	OnDescriptionChange(fn func())
}
