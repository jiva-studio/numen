package file

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNoteUnreadable is the error of a note read again whose file could not be read.
var ErrNoteUnreadable = errors.New("the note could not be read")

// ReadNoteAgain reads a note of a vault into the index again.
type ReadNoteAgain struct {
	refresh func(ctx context.Context, v domain.Vault, paths []string) (unreadable []string, err error)
}

// NewReadNoteAgain returns the use case over refresh, which reads the notes at paths and returns the paths
// it could not read.
func NewReadNoteAgain(
	refresh func(ctx context.Context, v domain.Vault, paths []string) (unreadable []string, err error),
) *ReadNoteAgain {
	return &ReadNoteAgain{refresh: refresh}
}

// Execute returns an error that wraps ErrNoteUnreadable when the file at path could not be read.
func (u *ReadNoteAgain) Execute(ctx context.Context, v domain.Vault, path string) error {
	unreadable, err := u.refresh(ctx, v, []string{path})
	if err != nil {
		return err
	}
	if slices.Contains(unreadable, path) {
		return fmt.Errorf("%s: %w", path, ErrNoteUnreadable)
	}
	return nil
}
