package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Window is what a person is looking at.
//
// It is how something working the vault beside them puts a place in front of
// them. Only an application with a window can satisfy it; where there is none,
// nothing asks.
type Window interface {
	// Focus puts this place in front of the person: a note becomes the one the
	// neighbourhood is seen from, and a document is opened where the place
	// says. A place already in focus is focused again, which is what asking for
	// it means.
	Focus(ctx context.Context, at domain.Place) error

	// Moved says a note is no longer where it was. Whoever is showing it at the
	// name it had follows it to the name it now has.
	Moved(ctx context.Context, went domain.Move) error

	// Editing says a change to a note's prose is being made, so that a person
	// reading that note sees it arrive where it belongs.
	Editing(ctx context.Context, said domain.Edit) error
}
