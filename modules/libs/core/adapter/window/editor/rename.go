package editor

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// RenameNote gives a note a different name.
func (h *noteHandler) RenameNote(
	ctx context.Context, r *connect.Request[v1.RenameNoteRequest],
) (*connect.Response[v1.RenameNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	renamed, err := h.rename.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTitle())
	out := &v1.RenameNoteResponse{
		Path:       renamed.Path,
		Title:      renamed.Title,
		NameSource: newWireNameSource(renamed.By),
	}
	if renamed.Moved != nil {
		out.Moved = newMoveResult(*renamed.Moved)
	}
	out.IsUnlevelled = h.state.isUnlevelled(err)
	if err != nil && !out.GetIsUnlevelled() {
		reason, ok := wire.GetErrorCode(err)
		switch {
		case ok:
			out.Error = &reason
		case out.GetPath() == "":
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// The note was opened and its name was written, so what stands is the
		// answer. A rename that came apart afterwards is a rename that happened.
	}
	return connect.NewResponse(out), nil
}

// newWireNameSource is which of the two a rename wrote, as the schema carries it.
func newWireNameSource(delta note.NameSource) v1.NameSource {
	switch delta {
	case note.ByFrontmatter:
		return v1.NameSource_NAME_SOURCE_FRONTMATTER
	case note.ByFilename:
		return v1.NameSource_NAME_SOURCE_FILENAME
	default:
		return v1.NameSource_NAME_SOURCE_UNSPECIFIED
	}
}

// newMoveResult is what the file did, as the schema carries it.
func newMoveResult(moved note.MoveResult) *v1.MoveResult {
	return &v1.MoveResult{
		From: moved.From, To: moved.To,
		Repaired: moved.Repaired, Dangling: moved.Dangling,
	}
}
