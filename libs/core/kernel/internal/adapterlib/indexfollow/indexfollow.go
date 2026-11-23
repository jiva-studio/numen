// Package indexfollow tells a write that reached the vault while the index did not follow from one that failed.
package indexfollow

import (
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/edit"
)

// Follow reads the error of a write.
type Follow struct{}

// IsIndexOutdated reports whether err says the write reached the vault and the index did not follow.
func (Follow) IsIndexOutdated(err error) bool { return errors.Is(err, edit.ErrIndexOutdated) }
