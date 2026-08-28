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

	// The folder says which of its entries are notes, and the index says what
	// each of those notes is. A listing carries both.
	types, err := a.typesOf(ctx, showing, held)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.ListResponse{Entries: make([]*v1.Entry, 0, len(held))}
	for _, entry := range held {
		out.Entries = append(out.Entries, &v1.Entry{
			Path:   entry.Path,
			Name:   entry.Name,
			Folder: entry.Folder,
			Kind:   kindOf(entry.Kind),
			Type:   typeOf(types[entry.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// typesOf is what each note of a listing is, keyed by the path it is filed
// under.
func (a *API) typesOf(ctx context.Context, showing domain.Vault, held []domain.Entry) (map[string]domain.NoteType, error) {
	paths := make([]string, 0, len(held))
	for _, entry := range held {
		if !entry.Folder && entry.Kind == domain.KindNote {
			paths = append(paths, entry.Path)
		}
	}
	return a.typesAt(ctx, showing, paths)
}

// typesAt is what each of the notes at those paths is, keyed by path. A build
// holding no index answers nothing, and every file is then drawn as the file it
// is.
func (a *API) typesAt(ctx context.Context, showing domain.Vault, paths []string) (map[string]domain.NoteType, error) {
	if a.Notes == nil || len(paths) == 0 {
		return nil, nil
	}
	return a.Notes.Types(ctx, showing.ID, paths)
}

// Standing hands the client what the vault holds at each of those paths, so
// that a client holding a path opens what stands there in the editor made for
// it.
//
// The kind is read off the vault and the type off the index, so a path nothing
// has scanned is answered with the source that stands there. A window standing
// on nothing holds no source, and a path with nothing at it is left out.
func (a *API) Standing(ctx context.Context, r *connect.Request[v1.StandingRequest]) (*connect.Response[v1.StandingResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		return connect.NewResponse(&v1.StandingResponse{}), nil
	}
	reader, err := a.Readers.Open(showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// In the order they were asked about, and the notes among them kept to ask
	// the index about in one question.
	paths := eachOnce(r.Msg.GetPaths())
	out := &v1.StandingResponse{Found: make([]*v1.Standing, 0, len(paths))}
	notes := make([]string, 0, len(paths))
	for _, path := range paths {
		ref, err := reader.Stat(ctx, path)
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, port.ErrOutside) {
			continue
		}
		// A file the vault leaves alone stands there and is no source, which is
		// the kind a zero reference carries.
		if err != nil && !errors.Is(err, port.ErrNotANote) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Found = append(out.Found, &v1.Standing{Path: path, Kind: kindOf(ref.Kind)})
		if ref.Kind == domain.KindNote {
			notes = append(notes, path)
		}
	}

	types, err := a.typesAt(ctx, showing, notes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	for _, one := range out.GetFound() {
		one.Type = typeOf(types[one.GetPath()])
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

// typeOf is which of three a note is, as the schema carries it. A note carrying
// no type of its own is an ordinary note.
func typeOf(noteType domain.NoteType) v1.NoteType {
	switch noteType {
	case domain.TypeDeck:
		return v1.NoteType_NOTE_TYPE_DECK
	case domain.TypeStencil:
		return v1.NoteType_NOTE_TYPE_STENCIL
	default:
		return v1.NoteType_NOTE_TYPE_UNSPECIFIED
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
