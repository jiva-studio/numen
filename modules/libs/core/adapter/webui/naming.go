package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// RenameNote gives a note a different name.
func (a *API) RenameNote(
	ctx context.Context, r *connect.Request[v1.RenameNoteRequest],
) (*connect.Response[v1.RenameNoteResponse], error) {
	if a.Notes.Rename == nil {
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

	renamed, err := a.Notes.Rename.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTitle())
	out := &v1.RenameNoteResponse{
		Path:  renamed.Path,
		Title: renamed.Title,
		By:    namingOf(renamed.By),
	}
	if renamed.Moved != nil {
		out.Moved = movedOf(*renamed.Moved)
	}
	if err != nil {
		reason, refused := wire.RefusalBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Refusal = &reason
	}
	return connect.NewResponse(out), nil
}

// errNoSettings is what a build that configures nothing answers.
var errNoSettings = errors.New("this build cannot turn settings")

// RemoveFile takes a file or a folder out of the vault. It goes to the trash,
// and a request that says so destroys a note.
func (a *API) RemoveFile(
	ctx context.Context, r *connect.Request[v1.RemoveFileRequest],
) (*connect.Response[v1.RemoveFileResponse], error) {
	if a.Notes.Remove == nil {
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

	removed, err := a.removal(ctx, showing, r.Msg.GetPath(), r.Msg.GetDestroy())
	if err != nil {
		reason, refused := wire.RefusalBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveFileResponse{Refusal: &reason}), nil
	}
	// The watcher reports only the paths the vault holds a source for. What
	// went is said here, so the tree drops the row whatever stood on it.
	a.Listeners.tell(change{paths: []string{removed.Path}})
	return connect.NewResponse(&v1.RemoveFileResponse{
		Trashed:  removed.Trashed,
		Dangling: removed.Dangling,
	}), nil
}

// removal is the two ways something leaves the vault.
func (a *API) removal(
	ctx context.Context,
	v domain.Vault,
	path string,
	destroy bool,
) (note.RemoveResult, error) {
	if destroy {
		return a.Notes.Remove.Destroy(ctx, v, path)
	}
	return a.Notes.Remove.Execute(ctx, v, path)
}

// namingOf is which of the three a rename wrote, as the schema carries it.
func namingOf(by note.NamedBy) v1.Naming {
	switch by {
	case note.ByFrontmatter:
		return v1.Naming_NAMING_FRONTMATTER
	case note.ByFilename:
		return v1.Naming_NAMING_FILENAME
	default:
		return v1.Naming_NAMING_UNSPECIFIED
	}
}

// movedOf is what the file did, as the schema carries it.
func movedOf(moved note.MoveResult) *v1.MoveResult {
	return &v1.MoveResult{From: moved.From, To: moved.To, Repaired: moved.Repaired}
}
