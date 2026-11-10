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

// EmitFileEvent tells the plugins of the window that an event happened.
func (e *Events) EmitFileEvent(event string) {
	if e.Emit != nil {
		e.Emit(event)
	}
}

// EmitVectorsReady tells the plugins of the window that every vector of the vault exists.
func (e *Events) EmitVectorsReady(_ context.Context, v domain.Vault) {
	e.EmitFileEvent(protocol.GetVectorsReadyEvent(string(v.ID)))
}
