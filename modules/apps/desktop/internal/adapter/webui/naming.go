package webui

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Rename gives a note a different name.
func (a *API) Rename(ctx context.Context, r *connect.Request[v1.RenameRequest]) (*connect.Response[v1.RenameResponse], error) {
	if a.Renames == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	renamed, err := a.Renames.Execute(ctx, a.Showing(), r.Msg.GetPath(), r.Msg.GetTitle())
	out := &v1.RenameResponse{
		Path:  renamed.Path,
		Title: renamed.Title,
		By:    namingOf(renamed.By),
	}
	if renamed.Moved != nil {
		out.Moved = movedOf(*renamed.Moved)
	}
	if errors.Is(err, port.ErrChanged) {
		out.Changed = true
		return connect.NewResponse(out), nil
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

// Syncing says whether a note's title and its filename are kept as one name.
func (a *API) Syncing(
	_ context.Context, _ *connect.Request[v1.SyncingRequest],
) (*connect.Response[v1.SyncingResponse], error) {
	return connect.NewResponse(&v1.SyncingResponse{
		SyncTitleAndFilename: bool(a.Sync.Kept()),
	}), nil
}

// ChooseSyncing writes that setting into the file a person configures this
// installation in.
func (a *API) ChooseSyncing(
	_ context.Context, r *connect.Request[v1.ChooseSyncingRequest],
) (*connect.Response[v1.ChooseSyncingResponse], error) {
	if a.Chooses == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoSettings)
	}
	if err := a.Chooses(note.Sync(r.Msg.GetSyncTitleAndFilename())); err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ChooseSyncingResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.ChooseSyncingResponse{}), nil
}

// errNoSettings is what a build that configures nothing answers.
var errNoSettings = errors.New("this build cannot turn settings")

// Remove takes a file or a folder out of the vault. It goes to the trash, and
// a request that says so destroys a note.
func (a *API) Remove(ctx context.Context, r *connect.Request[v1.RemoveRequest]) (*connect.Response[v1.RemoveResponse], error) {
	if a.Removes == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	removed, err := a.removal(ctx, r.Msg.GetPath(), r.Msg.GetDestroy())
	if err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.RemoveResponse{
		Trashed:  removed.Trashed,
		Dangling: removed.Dangling,
	}), nil
}

// removal is the two ways something leaves the vault.
func (a *API) removal(ctx context.Context, path string, destroy bool) (note.Removed, error) {
	if destroy {
		return a.Removes.Destroy(ctx, a.Showing(), path)
	}
	return a.Removes.Execute(ctx, a.Showing(), path)
}

// namingOf is which of the three a rename wrote, as the schema carries it.
func namingOf(by note.Naming) v1.Naming {
	switch by {
	case note.ByFrontmatter:
		return v1.Naming_NAMING_FRONTMATTER
	case note.ByHeading:
		return v1.Naming_NAMING_HEADING
	case note.ByFilename:
		return v1.Naming_NAMING_FILENAME
	default:
		return v1.Naming_NAMING_UNSPECIFIED
	}
}

// movedOf is what the file did, as the schema carries it.
func movedOf(moved note.Moved) *v1.Moved {
	return &v1.Moved{From: moved.From, To: moved.To, Repaired: moved.Repaired}
}
