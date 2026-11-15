package editor

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/protocol"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Events tells the plugins of the window that something happened, when the window has plugins.
type Events struct {
	Emit func(event string)
}

// EmitIndexChanged tells the plugins of the window that indexing changed the index of the vault.
func (e *Events) EmitIndexChanged(_ context.Context, v domain.Vault) {
	if e.Emit != nil {
		e.Emit(protocol.GetIndexChangedEvent(string(v.ID)))
	}
}
