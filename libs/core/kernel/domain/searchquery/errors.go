package searchquery

import "errors"

var (
	// ErrLimitExceeded is a query that exceeds a limit other than those of its text.
	ErrLimitExceeded = errors.New("the query exceeds a limit")
	// ErrNoModel is a query that uses meaning in an installation with no embedding model.
	ErrNoModel = errors.New("the query uses meaning, and the installation has no embedding model")
)
