package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/ulid"
)

// TellEdit is told what a write is doing while it is being made, for whoever is
// looking at the note. Nothing is told where nobody is drawing.
type TellEdit func(ctx context.Context, said domain.Edit)

// beginChange names one change and says what it is about to do. What comes back
//
// The name is a ULID, so the change the writer is telling about is named on the
// clock the write itself is stamped from.
func (tell TellEdit) beginChange(ctx context.Context, now port.Clock, response domain.Edit) func() {
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
