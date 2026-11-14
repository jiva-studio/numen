// Package window reads and moves what the user's window shows.
package window

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// inferredTabID is the identifier of the tab inferred from the focus of a call.
const inferredTabID = "front-tab"

// activationBound is the time the host waits for the window to show an activated tab. The host
// does not send its bound, so the text of the error names the default of the host.
const activationBound = "2 s"

// errMissingTabID is returned when a tab is activated without an identifier.
var errMissingTabID = errors.New("tabId is required")

// Port is the window of the host. A nil Port means the host offers none.
type Port interface {
	ReadOpenTabs(ctx context.Context) ([]host.OpenTab, string, error)
	ActivateTab(ctx context.Context, id string) (host.OpenTab, string, error)
	FocusNote(ctx context.Context, path string) (host.FocusedNote, error)
	FocusFile(ctx context.Context, path string, spans []host.ByteSpan) error
}

// Control reads and moves the window.
type Control struct {
	port Port
}

// NewControl creates a control over the window; port may be nil.
func NewControl(port Port) Control {
	return Control{port: port}
}

// ListTabs returns the open tabs and the identifier of the active one. A window with no tab, or a
// host with no window, yields the file in focus as the only tab when focus is not empty.
func (c Control) ListTabs(ctx context.Context, focus string) ([]host.OpenTab, string, error) {
	tabs, activeID, err := c.readTabs(ctx)
	if err != nil {
		return nil, "", err
	}
	if len(tabs) == 0 && focus != "" {
		inferred := host.OpenTab{ID: inferredTabID, Kind: "", Path: focus, Title: getTitle(focus), Place: nil}
		return []host.OpenTab{inferred}, inferred.ID, nil
	}
	return tabs, activeID, nil
}

func (c Control) readTabs(ctx context.Context) ([]host.OpenTab, string, error) {
	if c.port == nil {
		return nil, "", nil
	}
	tabs, activeID, err := c.port.ReadOpenTabs(ctx)
	if errors.Is(err, host.ErrNoWindow) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return tabs, activeID, nil
}

// ActivateTab makes the open tab with the identifier the active tab and returns the tab with the
// identifier of the active tab.
func (c Control) ActivateTab(ctx context.Context, id string) (host.OpenTab, string, error) {
	if id == "" {
		return host.OpenTab{}, "", errMissingTabID
	}
	if c.port == nil {
		return host.OpenTab{}, "", host.ErrNoWindow
	}
	open, _, err := c.port.ReadOpenTabs(ctx)
	if err != nil {
		return host.OpenTab{}, "", err
	}
	if !slices.ContainsFunc(open, func(tab host.OpenTab) bool { return tab.ID == id }) {
		return host.OpenTab{}, "", fmt.Errorf("no open tab has id %q", id)
	}
	tab, activeID, err := c.port.ActivateTab(ctx, id)
	if errors.Is(err, host.ErrNotExist) {
		return host.OpenTab{}, "", fmt.Errorf("no open tab has id %q", id)
	}
	if errors.Is(err, host.ErrTabNotShown) {
		return host.OpenTab{}, "", fmt.Errorf("the window did not show tab %q within %s", id, activationBound)
	}
	if err != nil {
		return host.OpenTab{}, "", err
	}
	return tab, activeID, nil
}

// FocusNote makes the note at the path the current one.
func (c Control) FocusNote(ctx context.Context, notePath string) (host.FocusedNote, error) {
	if c.port == nil {
		return host.FocusedNote{}, host.ErrNoWindow
	}
	return c.port.FocusNote(ctx, notePath)
}

// FocusFile opens the file at the first passage and highlights every passage. It returns the
// number of highlighted passages, which is 0 for a note, which opens whole.
func (c Control) FocusFile(ctx context.Context, filePath string, spans []host.ByteSpan) (int, error) {
	if c.port == nil {
		return 0, host.ErrNoWindow
	}
	if err := c.port.FocusFile(ctx, filePath, spans); err != nil {
		return 0, err
	}
	if isNotePath(filePath) {
		return 0, nil
	}
	return len(spans), nil
}

func isNotePath(filePath string) bool {
	return strings.EqualFold(path.Ext(filePath), domain.NoteExtension)
}

func getTitle(filePath string) string {
	base := path.Base(filePath)
	return strings.TrimSuffix(base, path.Ext(base))
}
