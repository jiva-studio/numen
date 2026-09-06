package wire

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Titled is what the vault calls the note at a path, which is the name a window
// shows beside the preset, the deck or the stencil that stands there.
//
// A build with no index, and a path the index holds no note at, are answered
// with no name: a window draws the path it already has.
func Titled(ctx context.Context, notes port.NoteQueries, vault domain.VaultID, path string) string {
	if notes == nil || path == "" {
		return ""
	}
	found, err := notes.Notes(ctx, vault, []string{path})
	if err != nil {
		return ""
	}
	return found[path].Title
}
