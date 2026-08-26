package webui

import (
	"context"
	"errors"
	"io/fs"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// List is what one folder of the vault holds. The vault settles the order the
// entries come in, and they cross in it.
func (a *API) List(ctx context.Context, r *connect.Request[v1.ListRequest]) (*connect.Response[v1.ListResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	held, err := reader.List(ctx, r.Msg.GetFolder())
	if err != nil {
		return nil, connect.NewError(listing(err), err)
	}

	out := &v1.ListResponse{Entries: make([]*v1.Entry, 0, len(held))}
	for _, entry := range held {
		out.Entries = append(out.Entries, &v1.Entry{
			Path:   entry.Path,
			Name:   entry.Name,
			Folder: entry.Folder,
			Kind:   kindOf(entry.Kind),
		})
	}
	return connect.NewResponse(out), nil
}

// Move puts a file or a folder somewhere else in the vault.
func (a *API) Move(ctx context.Context, r *connect.Request[v1.MoveRequest]) (*connect.Response[v1.MoveResponse], error) {
	if a.Moves == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	moved, err := a.Moves.Execute(ctx, showing, r.Msg.GetFrom(), r.Msg.GetTo())
	out := &v1.MoveResponse{}
	if moved.Landed {
		out.Moved = movedOf(moved)
	}
	if err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Refusal = &refusal
	}
	return connect.NewResponse(out), nil
}

// MakeFolder puts an empty folder in the vault, with the folders above it.
func (a *API) MakeFolder(ctx context.Context, r *connect.Request[v1.MakeFolderRequest]) (*connect.Response[v1.MakeFolderResponse], error) {
	if a.Writers == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	writer, err := a.Writers.Open(showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.MakeFolderResponse{}
	if err := writer.MakeFolder(ctx, r.Msg.GetPath()); err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Refusal = &refusal
	}
	return connect.NewResponse(out), nil
}

// listing is the code a folder that could not be listed is answered with.
func listing(err error) connect.Code {
	switch {
	case errors.Is(err, port.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, fs.ErrNotExist):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}

// kindOf is what the vault holds at a path, as the schema carries it. A file
// the vault holds no source for is unspecified.
func kindOf(kind domain.SourceKind) v1.SourceKind {
	switch kind {
	case domain.KindNote:
		return v1.SourceKind_SOURCE_KIND_NOTE
	case domain.KindBook:
		return v1.SourceKind_SOURCE_KIND_BOOK
	default:
		return v1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}
