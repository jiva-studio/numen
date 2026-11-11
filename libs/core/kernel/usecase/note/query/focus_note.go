package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// ErrNoNotePath is returned by a focus with no path.
var ErrNoNotePath = errors.New("note path is empty")

type FocusNote struct {
	queries noteport.NoteRefs
	view    PlaceFocus
}

// PlaceFocus asks the window to show a place.
type PlaceFocus interface {
	FocusPlace(ctx context.Context, pos domain.Place) error
}

func NewFocusNote(queries noteport.NoteRefs, view PlaceFocus) *FocusNote {
	return &FocusNote{queries: queries, view: view}
}

// Execute puts the note at path in focus and returns its reference. A path with no note in the
// index returns edit.ErrNotExist.
func (u *FocusNote) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteRef, error) {
	if path == "" {
		return domain.NoteRef{}, plugin.InvalidFocus(ErrNoNotePath)
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return domain.NoteRef{}, err
	}
	ref, recognized := found[path]
	if !recognized {
		return domain.NoteRef{}, fmt.Errorf("no note at %s: %w", path, edit.ErrNotExist)
	}
	if err := u.view.FocusPlace(ctx, domain.Place{Path: path}); err != nil {
		return domain.NoteRef{}, err
	}
	return ref, nil
}
