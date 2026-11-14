package indexport

// IndexFollow reports whether the index followed a write.
type IndexFollow interface {
	// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
	IsIndexOutdated(err error) bool
}
