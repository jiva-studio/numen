package note

import (
	"context"
	"crypto/rand"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/ulid"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// TellEdit reports what a write is doing while it runs, to viewers of the note.
// Nothing is reported when no viewer is drawing. The returned error is the
// failure to report.
type TellEdit func(ctx context.Context, edit domain.Edit) error

func (tell TellEdit) beginChange(ctx context.Context, now machineport.Clock, response domain.Edit) (func() error, error) {
	if tell == nil {
		return func() error { return nil }, nil
	}
	name, err := ulid.NewGenerator(rand.Reader).NewID(now.Now())
	if err != nil {
		return nil, err
	}
	response.Change = name
	if err := tell(ctx, response); err != nil {
		return nil, err
	}
	return func() error {
		return tell(ctx, domain.Edit{Change: name, Path: response.Path, IsDone: true})
	}, nil
}

// TellMove receives the destination of a moved note, for the viewer that shows it.
// A viewer of a renamed note holds a name with no file behind it, and later
// changes would be applied to a path the viewer does not show.
// It is not called when no viewer renders the note. The returned error is the
// failure to report.
type TellMove func(ctx context.Context, v domain.Vault, went domain.Move) error

// MoveView shows a window that a note moved.
type MoveView interface {
	ShowMove(ctx context.Context, went domain.Move) error
}

func NewTellMoveInShownVault(showing func() (domain.Vault, bool), view MoveView) TellMove {
	return func(ctx context.Context, v domain.Vault, went domain.Move) error {
		if current, isShowing := showing(); !isShowing || current.ID != v.ID {
			return nil
		}
		return view.ShowMove(ctx, went)
	}
}
