package editor

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"sync/atomic"
)

// PageAccess controls requests, writes and media served over the window's HTTP endpoints.
type PageAccess struct {
	vaults shownVault
	Gate   *wire.Gate
	Writes *wire.Writes
	Window *wire.Window
	Media  Media
}

// NewPageAccess creates the request and write gates for one window.
func NewPageAccess(tasks *run.Tasks, vaults shownVault) *PageAccess {
	window := &wire.Window{Name: wire.Editor, Tasks: tasks, Vault: func() string { return string(vaults.GetCurrentVault().ID) }}
	return &PageAccess{vaults: vaults, Gate: &wire.Gate{}, Writes: wire.NewWrites(window), Window: window}
}

// Media holds the loopback server that serves vault and plugin files.
type Media struct{ Playing atomic.Pointer[Loopback] }

// ClosePlaying stops the socket recordings are played over.
func (s *Media) ClosePlaying(ctx context.Context) error { return s.Playing.Load().Close(ctx) }

// GetMediaAddress is where the file with this stat is loaded from, and nowhere while the socket
// does not answer.
func (s *Media) GetMediaAddress(v domain.Vault, stat domain.FileStat) string {
	return s.Playing.Load().Address(v, stat)
}

// GetPluginFileAddress is where a file of a plugin's store of the vault is loaded from, and nowhere
// while the socket does not answer.
func (s *Media) GetPluginFileAddress(v domain.Vault, address file.Address, size int64) string {
	return s.Playing.Load().PluginFileAddress(v, address, size)
}
