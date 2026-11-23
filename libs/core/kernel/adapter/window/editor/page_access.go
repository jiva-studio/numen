package editor

import (
	"context"
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type pageSession interface {
	shownVault
	GetOpenTabs() window.OpenTabs
}

// PageAccess controls requests, writes and media served over the window's HTTP endpoints.
type PageAccess struct {
	session pageSession
	Gate    *wire.Gate
	Writes  *wire.Writes
	Window  *wire.Window
	Media   Media
}

// NewPageAccess creates the request and write gates for one window.
func NewPageAccess(tasks *run.Tasks, session pageSession) *PageAccess {
	window := &wire.Window{Name: wire.Editor, Tasks: tasks, GetVault: func() string { return string(session.GetCurrentVault().ID) }}
	return &PageAccess{session: session, Gate: &wire.Gate{}, Writes: wire.NewWrites(window), Window: window}
}

// Media holds the loopback server that serves vault and plugin files.
type Media struct{ playing atomic.Pointer[Loopback] }

// ClosePlaying stops the socket recordings are played over.
func (s *Media) ClosePlaying(ctx context.Context) error { return s.playing.Load().Close(ctx) }

// GetMediaAddress is where the file with this stat is loaded from, and nowhere while the socket
// does not answer.
func (s *Media) GetMediaAddress(v domain.Vault, stat domain.FileStat) string {
	return s.playing.Load().Address(v, stat)
}

// GetPluginFileAddress is where a file of a plugin's store of the vault is loaded from, and nowhere
// while the socket does not answer.
func (s *Media) GetPluginFileAddress(v domain.Vault, address file.Address, size int64) string {
	return s.playing.Load().PluginFileAddress(v, address, size)
}
