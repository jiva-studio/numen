package host

import (
	"context"
	"errors"
)

var (
	// ErrNoWindow is returned by a window call when the host has no window.
	ErrNoWindow = errors.New("host: the host has no window")
	// ErrTabNotShown is returned by ActivateTab when the window does not show the tab within the bound
	// of the host. It wraps the deadline error the host returns.
	ErrTabNotShown = errors.New("host: the window did not show the tab in time")
)

// FocusedNote is the note FocusNote made the current one.
type FocusedNote struct{ Path, Title, ID string }

// OpenTab is one tab of the window as it last reported it. Place is the JSON of the place its reader
// reported, in the shape of a contract such as contracts/readerplace, and empty for a tab whose kind
// reports none.
type OpenTab struct {
	ID, Kind, Path, Title, Place string
}

// PassageSpan is a byte range in a file's text version, as a passage search returns it. A passage
// address has the form numen:<path>?start=<bytes>&length=<bytes>: the path relative to the vault,
// percent-encoded as one path segment (url.PathEscape), and start and length the byte range of the
// passage in the file's text version.
type PassageSpan struct{ Start, Length int }

// Window is the main window of the plugin's vault. Every call needs window:focus. A path with no file
// in the vault returns ErrNotExist, and a host with no window returns ErrNoWindow.
type Window interface {
	// FocusNote makes the note at the path the current one, and the neighborhood view is centered on it.
	FocusNote(ctx context.Context, path string) (FocusedNote, error)
	// FocusPassages opens the file at the first passage and highlights every passage.
	FocusPassages(ctx context.Context, path string, passages []PassageSpan) error
	// ReadOpenTabs returns the tabs the window last reported, and the ID of the active one.
	ReadOpenTabs(ctx context.Context) (tabs []OpenTab, activeID string, err error)
	// ActivateTab makes the open tab with the id the active tab, and returns the tab and the active id
	// as the window reports them afterwards. An id with no tab in the window returns ErrNotExist, and a tab
	// the window does not show within the bound of the host returns ErrTabNotShown.
	ActivateTab(ctx context.Context, id string) (tab OpenTab, activeID string, err error)
}
