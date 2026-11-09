package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
)

// ErrNoNotePath is returned by a focus with no path.
var ErrNoNotePath = errors.New("note path is empty")

// FocusNote sets the focused note.
type FocusNote struct {
	queries noteport.NoteQueries
	view    windowport.Window
}

func NewFocusNote(queries noteport.NoteQueries, view windowport.Window) *FocusNote {
	return &FocusNote{queries: queries, view: view}
}

// Execute puts the note at path in focus and returns its reference. A path with no note in the
// index returns vaultport.ErrNotExist.
func (u *FocusNote) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteRef, error) {
	if path == "" {
		return domain.NoteRef{}, ErrNoNotePath
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return domain.NoteRef{}, err
	}
	ref, recognized := found[path]
	if !recognized {
		return domain.NoteRef{}, fmt.Errorf("no note at %s: %w", path, vaultport.ErrNotExist)
	}
	if err := u.view.FocusPlace(ctx, domain.Place{Path: path}); err != nil {
		return domain.NoteRef{}, err
	}
	return ref, nil
}
