package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// Viewing is this window, for whatever asks for a note to be put in front of
// the person.
func (a *API) Viewing() port.View { return viewing{a} }

// viewing tells the clients and nothing more. What travelling there looks
// like is theirs, and a note asked for while nobody is drawing is a note
// nobody sees.
type viewing struct{ *API }

func (v viewing) Focus(_ context.Context, path string) error {
	v.Watching.tell(path)
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
		case path, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.FocusResponse{Path: path}); err != nil {
				return err
			}
		}
	}
}
