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

// EmitSearchSourceReady tells the plugins of the window that the search source has indexed the vault.
func (e *Events) EmitSearchSourceReady(_ context.Context, v domain.Vault) {
	if e.Emit != nil {
		e.Emit(protocol.GetSearchSourceReadyEvent(string(v.ID)))
	}
}
