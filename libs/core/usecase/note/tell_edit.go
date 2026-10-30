package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/ulid"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// TellEdit reports what a write is doing while it runs, to viewers of the note.
// Nothing is reported when no viewer is drawing. The returned error is the
// failure to report.
type TellEdit func(ctx context.Context, edit domain.Edit) error

// beginChange names one change and reports its start. It returns the function
// that reports its end.
//
// The name is a ULID, so the change is named on the same clock the write is
// stamped from.
func (tell TellEdit) beginChange(ctx context.Context, now machineport.Clock, response domain.Edit) (func() error, error) {
	if tell == nil {
		return func() error { return nil }, nil
	}
	name := ulid.New(now())
	response.Change = name
	if err := tell(ctx, response); err != nil {
		return nil, err
	}
	return func() error {
		return tell(ctx, domain.Edit{Change: name, Path: response.Path, IsDone: true})
	}, nil
}

// TellMove receives the destination of a moved note, for the viewer that shows
// it.
//
// A viewer of a renamed note holds a name with no file behind it, and every
// later change to the note is applied to a path the viewer does not show.
// TellMove is not called when no viewer renders the note. The returned error is
// the failure to report.
type TellMove func(ctx context.Context, went domain.Move) error
