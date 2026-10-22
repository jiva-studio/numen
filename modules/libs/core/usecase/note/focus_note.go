package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// FocusNote makes a note the one the person is looking at.
type FocusNote struct {
	queries port.NoteQueries
	view    port.Window
}

func NewFocusNote(queries port.NoteQueries, view port.Window) *FocusNote {
	return &FocusNote{queries: queries, view: view}
}

// Execute puts the note at path in focus and returns what it is called. A path
// naming no note is refused.
func (u *FocusNote) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteRef, error) {
	if path == "" {
		return domain.NoteRef{}, errors.New("name the note to put in focus")
	}
	found, err := u.queries.GetNotes(ctx, v.ID, []string{path})
	if err != nil {
		return domain.NoteRef{}, err
	}
	ref, recognized := found[path]
	if !recognized {
		return domain.NoteRef{}, fmt.Errorf("no note at %s", path)
	}
	if err := u.view.FocusPlace(ctx, domain.Place{Path: path}); err != nil {
		return domain.NoteRef{}, err
	}
	return ref, nil
}
