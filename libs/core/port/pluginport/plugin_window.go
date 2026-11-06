package pluginport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var (
	// ErrNoWindow is what a window call gets from a host that shows no window.
	ErrNoWindow = errors.New("the host has no window")
	// ErrInvalidFocus is what a focus call gets for an argument the use case refuses: an empty path,
	// a passage that starts before the text or is empty, or too many passages. The error wraps it
	// with the use case's message.
	ErrInvalidFocus = errors.New("the place cannot be shown")
)

// PluginWindow is the main window of the vault, as the plugins that call the host move it and read
// its tabs. A path the vault holds nothing at is fs.ErrNotExist.
type PluginWindow interface {
	// FocusNote makes the note at the path the current one and returns it.
	FocusNote(ctx context.Context, path string) (domain.NoteRef, error)
	// FocusPassages opens the file at the path at the first passage and highlights every passage. No
	// passages opens the file at its beginning.
	FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error
	// ReadOpenTabs is the tabs the window last reported, and the active one.
	ReadOpenTabs(ctx context.Context) (domain.OpenTabs, error)
}

// PluginWindowHost receives the window that plugins move.
type PluginWindowHost interface {
	// SetPluginWindow is the main window the plugins move. Nil is none.
	SetPluginWindow(window PluginWindow)
}
