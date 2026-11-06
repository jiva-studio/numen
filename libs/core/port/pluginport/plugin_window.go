package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrNoWindow is returned by a window call when the host shows no window.
	ErrNoWindow = errors.New("the host has no window")
	// ErrInvalidFocus is returned by a focus call for an argument the use case rejects: an empty path,
	// a passage that starts before the text or is empty, or too many passages. The error wraps it
	// with the use case's message.
	ErrInvalidFocus = errors.New("invalid focus target")
	// ErrTabNotShown is returned by ActivateTab when the window does not report the tab active
	// within the bound of the call. The error wraps it with the id of the tab.
	ErrTabNotShown = errors.New("the window did not show the tab in time")
)

// PluginWindow is the main window of the vault, exposed to the plugins that call the host. A path
// with no file in the vault returns fs.ErrNotExist.
type PluginWindow interface {
	// FocusNote makes the note at the path the current one and returns it.
	FocusNote(ctx context.Context, path string) (domain.NoteRef, error)
	// FocusPassages opens the file at the path at the first passage and highlights every passage. No
	// passages opens the file at its beginning.
	FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error
	// ReadOpenTabs returns the tabs the window last reported, and the active one.
	ReadOpenTabs(ctx context.Context) (domain.OpenTabs, error)
	// ActivateTab makes the open tab with the id the active tab, and returns the open tabs as the
	// window reports them afterwards. An id with no open tab returns fs.ErrNotExist, and a tab the
	// window does not report active in time returns ErrTabNotShown.
	ActivateTab(ctx context.Context, id string) (domain.OpenTabs, error)
}

// PluginWindowHost receives the window exposed to plugins.
type PluginWindowHost interface {
	// SetPluginWindow sets the window exposed to plugins. Nil means none.
	SetPluginWindow(window PluginWindow)
}
