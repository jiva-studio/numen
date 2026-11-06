package host

import (
	"context"
	"errors"
)

// ErrNoWindow is what a window call gets from a host with no window.
var ErrNoWindow = errors.New("host: the host has no window")

// FocusedNote is the note FocusNote made the current one.
type FocusedNote struct{ Path, Title, ID string }

// OpenTab is one tab of the window as it last reported it. State is the JSON its kind saved: for a
// reader tab, the selector of its place ({"page":N} PDF, {"cfi":"..."} EPUB, {"t":[start,end]} seconds of a recording).
type OpenTab struct{ ID, Kind, Path, Title, State string }

// PassageSpan is a run of a file's text version in bytes, as library_search returns it.
type PassageSpan struct{ Start, Length int }

// Window is the window of the vault a plugin works. Every call needs window:focus. A path the vault
// holds nothing at is ErrNotExist, and a host with no window answers ErrNoWindow.
type Window interface {
	// FocusNote makes the note at the path the current one, and the neighbourhood view is drawn around it.
	FocusNote(ctx context.Context, path string) (FocusedNote, error)
	// FocusPassages opens the file at the first passage and highlights every passage.
	FocusPassages(ctx context.Context, path string, passages []PassageSpan) error
	// ReadOpenTabs is the tabs the window last reported, and the active one.
	ReadOpenTabs(ctx context.Context) (tabs []OpenTab, activeID string, err error)
}
