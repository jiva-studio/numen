package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/ulid"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// TellEdit reports what a write is doing while it runs, to viewers of the note.
// Nothing is reported when no viewer is drawing.
type TellEdit func(ctx context.Context, edit domain.Edit)

// beginChange names one change and reports its start. It returns the function
// that reports its end.
//
// The name is a ULID, so the change is named on the same clock the write is
// stamped from.
func (tell TellEdit) beginChange(ctx context.Context, now machineport.Clock, response domain.Edit) func() {
	if tell == nil {
		return func() {}
	}
	name := ulid.New(now())
	response.Change = name
	tell(ctx, response)
	return func() {
		tell(ctx, domain.Edit{Change: name, Path: response.Path, IsDone: true})
	}
}
