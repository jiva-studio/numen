package editor

import (
	"context"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Viewing is this window, for whatever asks for a place to be put in front of
// the person.
func (a *API) Viewing() port.Window { return view{a} }

// view tells the clients and nothing more. What travelling there looks
// like is theirs, and a place asked for while nobody is drawing is a place
// nobody sees.
type view struct{ *API }

func (v view) Focus(_ context.Context, at domain.Place) error {
	v.Places.tell(at)
	return nil
}

// WatchFocus reports what something else asked to be put in front of the
// person, for as long as the client listens.
func (a *API) WatchFocus(
	ctx context.Context,
	_ *connect.Request[v1.WatchFocusRequest],
	out *connect.ServerStream[v1.WatchFocusResponse],
) error {
	line, done := a.Places.listen()
	defer done()

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// A place nobody asked for names nothing, and is here to fail when
			// the client has gone.
			if err := out.Send(&v1.WatchFocusResponse{}); err != nil {
				return err
			}
		case at, open := <-line:
			if !open {
				return nil
			}
			also := make([]*v1.Stretch, 0, len(at.Stretches))
			for _, one := range at.Stretches {
				also = append(also, &v1.Stretch{Start: int32(one.Start), Length: int32(one.Length)})
			}
			if err := out.Send(&v1.WatchFocusResponse{
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

func (v view) Moved(_ context.Context, went domain.Move) error {
	v.Listeners.tell(change{renamed: []domain.Move{went}})
	return nil
}

func (v view) Editing(_ context.Context, said domain.Edit) error {
	v.Edits.tell(said)
	return nil
}

// WatchEdits reports a change being made to a note's prose while it is being
// made, for as long as the client listens.
func (a *API) WatchEdits(
	ctx context.Context,
	_ *connect.Request[v1.WatchEditsRequest],
	out *connect.ServerStream[v1.WatchEditsResponse],
) error {
	line, done := a.Edits.listen()
	defer done()

	// A stream that says nothing until a note is changed is indistinguishable
	// from one that never opened.
	if err := out.Send(&v1.WatchEditsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchEditsResponse{}); err != nil {
				return err
			}
		case said, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchEditsResponse{
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
