package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// View is what a person is looking at.
//
// It is how something working the vault beside them puts a note in front of
// them. Only an application with a window can satisfy it; where there is none,
// nothing asks.
type View interface {
	// Focus makes this note the one the neighbourhood is seen from. A note
	// already in focus is focused again, which is what asking for it means.
	Focus(ctx context.Context, path string) error

	// Editing says a change to a note's prose is being made, so that a person
	// reading that note sees it arrive where it belongs.
	Editing(ctx context.Context, said domain.Editing) error
}
