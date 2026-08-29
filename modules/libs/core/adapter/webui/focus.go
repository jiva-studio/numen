package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Viewing is this window, for whatever asks for a place to be put in front of
// the person.
func (a *API) Viewing() port.View { return viewing{a} }

// viewing tells the clients and nothing more. What travelling there looks
// like is theirs, and a place asked for while nobody is drawing is a place
// nobody sees.
type viewing struct{ *API }

func (v viewing) Focus(_ context.Context, at domain.Place) error {
	v.Watching.tell(at)
	return nil
}

// Focus reports what something else asked to be put in front of the person,
// for as long as the client listens.
func (a *API) Focus(
	ctx context.Context,
	_ *connect.Request[v1.FocusRequest],
	out *connect.ServerStream[v1.FocusResponse],
) error {
	line, done := a.Watching.listen()
	defer done()

	for {
		select {
		case <-ctx.Done():
			return nil
		case at, open := <-line:
			if !open {
				return nil
			}
			also := make([]*v1.Stretch, 0, len(at.Also))
			for _, one := range at.Also {
				also = append(also, &v1.Stretch{Start: int32(one.Start), Length: int32(one.Length)})
			}
			if err := out.Send(&v1.FocusResponse{
				Path:   at.Path,
				Start:  int32(at.Start),
				Length: int32(at.Length),
				Also:   also,
			}); err != nil {
				return err
			}
		}
	}
}

func (v viewing) Moved(_ context.Context, went domain.Went) error {
	v.Listeners.tell(changed{renamed: []domain.Went{went}})
	return nil
}

func (v viewing) Editing(_ context.Context, said domain.Editing) error {
	v.Drawing.tell(said)
	return nil
}

// Editing reports a change being made to a note's prose while it is being made,
// for as long as the client listens.
func (a *API) Editing(
	ctx context.Context,
	_ *connect.Request[v1.EditingRequest],
	out *connect.ServerStream[v1.EditingResponse],
) error {
	line, done := a.Drawing.listen()
	defer done()

	// A stream that says nothing until a note is changed is indistinguishable
	// from one that never opened.
	if err := out.Send(&v1.EditingResponse{}); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case said, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.EditingResponse{
				Change: said.Change,
				Path:   said.Path,
				From:   int32(said.From),
				To:     int32(said.To),
				Text:   said.Text,
				Done:   said.Done,
			}); err != nil {
				return err
			}
		}
	}
}
