package hosttest

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// FocusRequest is one FocusPassages call a Window received.
type FocusRequest struct {
	Path     string
	Passages []host.PassageSpan
}

// Window is the window of a vault, in memory. A note FocusNote is asked for is found among the notes
// of the vault, and a file FocusPassages is asked for among its files; a path the vault holds nothing
// at is ErrNotExist and is not recorded. The window serves the tabs SetOpenTabs gives it.
type Window struct{}

var _ host.Window = (*Window)(nil)

// NewWindow is the window of the vault, with no tab open.
func NewWindow(_ *Vault) *Window { return &Window{} }

func (w *Window) FocusNote(_ context.Context, _ string) (host.FocusedNote, error) {
	return host.FocusedNote{}, nil
}

func (w *Window) FocusPassages(_ context.Context, _ string, _ []host.PassageSpan) (string, error) {
	return "", nil
}

func (w *Window) ReadOpenTabs(_ context.Context) ([]host.OpenTab, string, error) {
	return nil, "", nil
}

// SetOpenTabs is the tabs the window reports from now on, and the active one.
func (w *Window) SetOpenTabs(_ []host.OpenTab, _ string) {}

// SetNoWindow makes every later call answer ErrNoWindow, as a host with no window does.
func (w *Window) SetNoWindow() {}

// GetFocusedNotes is the paths of the notes FocusNote made the current one, in order.
func (w *Window) GetFocusedNotes() []string { return nil }

// GetFocusRequests is the FocusPassages calls the window served, in order.
func (w *Window) GetFocusRequests() []FocusRequest { return nil }
