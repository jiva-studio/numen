package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// errNoEditing is what a build with no write path answers.
var errNoEditing = errors.New("this build cannot edit notes")

// Read hands the client the prose of a note.
func (a *API) Read(ctx context.Context, r *connect.Request[v1.ReadRequest]) (*connect.Response[v1.ReadResponse], error) {
	if a.Reads == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	found, err := a.Reads.Execute(ctx, a.Vault, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ReadResponse{Body: found.Body}
	if refusal, refused := refusalOf(found.Outcome); refused {
		out.Refusal = &refusal
	}
	return connect.NewResponse(out), nil
}

// Write puts prose into a note. What is on disk is replaced.
func (a *API) Write(ctx context.Context, r *connect.Request[v1.WriteRequest]) (*connect.Response[v1.WriteResponse], error) {
	if a.Saves == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	// Counted around the write and not around the answer, so that closing waits
	// for what reaches the vault.
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	err := a.Saves.Save(ctx, a.Vault, r.Msg.GetPath(), r.Msg.GetBody())
	a.Writing.done()
	if err == nil {
		return connect.NewResponse(&v1.WriteResponse{}), nil
	}
	refusal, refused := refusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteResponse{Refusal: &refusal}), nil
}

// refusedBy says which refusal a write's error is, and whether it is one at all.
// Anything else is the vault being out of reach.
func refusedBy(err error) (v1.Refusal, bool) {
	switch {
	case errors.Is(err, note.ErrTooLarge):
		return v1.Refusal_REFUSAL_TOO_LARGE, true
	case errors.Is(err, note.ErrUnreadable):
		return v1.Refusal_REFUSAL_UNREADABLE, true
	case errors.Is(err, note.ErrBodyRefused):
		return v1.Refusal_REFUSAL_BODY_REFUSED, true
	case errors.Is(err, port.ErrNotANote):
		return v1.Refusal_REFUSAL_NOT_A_NOTE, true
	default:
		return v1.Refusal_REFUSAL_UNSPECIFIED, false
	}
}

// refusalOf says which refusal an outcome is, and whether it is one at all.
func refusalOf(o note.Outcome) (v1.Refusal, bool) {
	switch o {
	case note.Missing:
		return v1.Refusal_REFUSAL_MISSING, true
	case note.NotANote:
		return v1.Refusal_REFUSAL_NOT_A_NOTE, true
	case note.NotText:
		return v1.Refusal_REFUSAL_NOT_TEXT, true
	case note.TooLarge:
		return v1.Refusal_REFUSAL_TOO_LARGE, true
	case note.Unreadable:
		return v1.Refusal_REFUSAL_UNREADABLE, true
	default:
		return v1.Refusal_REFUSAL_UNSPECIFIED, false
	}
}
