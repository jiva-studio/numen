// Package text holds the records of the text versions and their sentinel errors.
package text

import "errors"

var (
	// ErrNoText is a file, a version or a key nothing has text for, or a
	// version whose file changed since it was recorded.
	ErrNoText        = errors.New("there is no such text")
	ErrNotTextPlugin = errors.New("the plugin makes no text")
	// ErrInvalidVersion is a version with no key, no kind or no file, a file
	// that is not there or is not JSON Lines of segments, or a source that is
	// no other version of the file.
	ErrInvalidVersion = errors.New("the version is not valid")
	// ErrFileUnreadable wraps the failure to stat, open or read the file whose hash is requested. A failure
	// of the vault, of the index or of the context is returned without it.
	ErrFileUnreadable = errors.New("the file cannot be read")
)
