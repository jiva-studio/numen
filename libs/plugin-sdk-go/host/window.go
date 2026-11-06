package host

import (
	"context"
	"errors"
)

// ErrNoWindow is what a window call gets from a host with no window.
var ErrNoWindow = errors.New("host: the host has no window")

// FocusedNote is the note FocusNote made the current one.
type FocusedNote struct{ Path, Title, ID string }

// OpenTab is one tab of the window as it last reported it. Place is the place its reader reported,
// nil for a tab whose kind reports none.
type OpenTab struct {
	ID, Kind, Path, Title string
	Place                 *TabPlace
}

// TabPlace is the place a reader tab shows, as its reader reports it. A zero field is a place the
// reader does not have.
type TabPlace struct {
	// Page is the page a PDF shows, from 1, and PageCount the number of its pages.
	Page, PageCount int
	// PositionSeconds is the time of a recording, in seconds from its start; nil for a tab that is
	// not a recording.
	PositionSeconds *float64
	// CFI is the EPUB CFI of the place shown, and Chapter the table-of-contents title there.
	CFI, Chapter string
}

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
	// ActivateTab makes the open tab with the id the active tab, and answers the tab and the active id
	// as the window reports them afterwards. An id the window has no tab with is ErrNotExist, and a tab
	// the window does not show within the bound of the host is the deadline error the host answers.
	ActivateTab(ctx context.Context, id string) (tab OpenTab, activeID string, err error)
}
