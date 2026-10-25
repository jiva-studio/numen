package artifacts

import "errors"

var (
	// ErrNotReady is a question about a vault whose index or store is not up yet.
	ErrNotReady = errors.New("the vault is still coming up")
)
