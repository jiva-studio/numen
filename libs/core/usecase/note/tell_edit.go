package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/ulid"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// TellEdit is told what a write is doing while it is being made, for whoever is
// looking at the note. Nothing is told where nobody is drawing. The error it
// returns is the failure to tell.
type TellEdit func(ctx context.Context, edit domain.Edit) error

// beginChange names one change and says what it is about to do. It returns the
// function that reports the change as finished.
//
// The name is a ULID, so the change the writer is telling about is named on the
// clock the write itself is stamped from.
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
