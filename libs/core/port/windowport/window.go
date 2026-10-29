package windowport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Window is the application window that shows places to the user.
//
// A background worker uses it to bring a place into view. Only an application
// with a window can implement it; without one, no caller requests it.
type Window interface {
	// FocusPlace shows this place: a note becomes the focus of the neighbourhood
	// view, and a document is opened at the position the place specifies. A place
	// already in focus is focused again.
	FocusPlace(ctx context.Context, pos domain.Place) error

	// ShowMove reports that a note moved. A view showing the note under its old
	// name follows it to the new name.
	ShowMove(ctx context.Context, went domain.Move) error

	// ShowEdit reports that a change to a note's prose is in progress, so a view of
	// that note shows the change at the right position.
	ShowEdit(ctx context.Context, response domain.Edit) error
}
