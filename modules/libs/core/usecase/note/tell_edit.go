package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// TellEdit is told what a write is doing while it is being made, for whoever is
// looking at the note. Nothing is told where nobody is drawing.
type TellEdit func(ctx context.Context, said domain.Edit)

// begins names one change and says what it is about to do. What comes back ends
// it, and ends it whether the change landed or was refused.
//
// The name is a ULID, so the change the writer is telling about is named on the
// clock the write itself is stamped from.
func (tell TellEdit) begins(ctx context.Context, now port.Clock, said domain.Edit) func() {
	if tell == nil {
		return func() {}
	}
	name, err := ulid.New(now())
	if err != nil {
		return func() {}
	}
	said.Change = name
	tell(ctx, said)
	return func() {
		tell(ctx, domain.Edit{Change: name, Path: said.Path, Done: true})
	}
}
