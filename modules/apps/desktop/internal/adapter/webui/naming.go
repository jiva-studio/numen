package webui

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// Rename gives a note a different name. Whichever of the title, the heading and
// the filename names the note is brought into line, and the file is renamed
// with it.
//
// The note is written before the file is moved, so a refused move answers with
// the name the note now carries and the path it still has.
func (a *API) Rename(ctx context.Context, r *connect.Request[v1.RenameRequest]) (*connect.Response[v1.RenameResponse], error) {
	if a.Renames == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	// A title that leaves no filename names nothing, and the note is not opened
	// for it.
	if name, _ := domain.Filename(r.Msg.GetTitle()); name == "" {
		refusal := v1.Refusal_REFUSAL_UNNAMEABLE
		return connect.NewResponse(&v1.RenameResponse{Refusal: &refusal}), nil
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	renamed, err := a.Renames.Execute(ctx, a.Vault, r.Msg.GetPath(), r.Msg.GetTitle())
	out := &v1.RenameResponse{
		Path:  renamed.Path,
		Title: renamed.Title,
		By:    namingOf(renamed.By),
	}
	if err != nil {
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Refusal = &refusal
		return connect.NewResponse(out), nil
	}
	if renamed.Moved != nil {
		out.Moved = movedOf(*renamed.Moved)
	}
	return connect.NewResponse(out), nil
}

// Remove takes a note out of the vault. It goes to the trash, and a request
// that says so destroys it.
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

// removal is the two ways a note leaves the vault.
func (a *API) removal(ctx context.Context, path string, destroy bool) (note.Removed, error) {
	if destroy {
		return a.Removes.Destroy(ctx, a.Vault, path)
	}
	return a.Removes.Execute(ctx, a.Vault, path)
}

// namingOf is which of the three a rename wrote, as the schema carries it.
func namingOf(by string) v1.Naming {
	switch by {
	case "frontmatter":
		return v1.Naming_NAMING_FRONTMATTER
	case "heading":
		return v1.Naming_NAMING_HEADING
	case "filename":
		return v1.Naming_NAMING_FILENAME
	default:
		return v1.Naming_NAMING_UNSPECIFIED
	}
}

// movedOf is what the file did, as the schema carries it. A retargeted link
// crosses as the value of the address it was written by, which is what names
// the link to the person.
func movedOf(moved note.Moved) *v1.Moved {
	out := &v1.Moved{From: moved.From, To: moved.To, Repaired: moved.Repaired}
	for _, one := range moved.Retargeted {
		out.Retargeted = append(out.Retargeted, &v1.Retargeted{
			In: one.In, Target: one.Target.Value, Now: one.Now,
		})
	}
	return out
}
