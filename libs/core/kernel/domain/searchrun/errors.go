package searchrun

import "errors"

var (
	// ErrNotFound is a search that is unknown, expired, removed or of another plugin.
	ErrNotFound = errors.New("no such search")
	// ErrTooManySearches is a start while the most searches kept are all running.
	ErrTooManySearches = errors.New("too many searches are kept")
	// ErrInvalidPageToken is a page token of another search or one that is not valid.
	ErrInvalidPageToken = errors.New("the page token is not valid for this search")
)
