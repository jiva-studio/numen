package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// TellMove receives the destination of a moved note, for the viewer that shows
// it.
//
// A viewer of a renamed note holds a name with no file behind it, and every
// later change to the note is applied to a path the viewer does not show.
// TellMove is not called when no viewer renders the note. The returned error is
// the failure to report.
type TellMove func(ctx context.Context, went domain.Move) error
