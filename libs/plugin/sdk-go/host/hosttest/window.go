package hosttest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// FocusRequest is one FocusFile call a Window received.
type FocusRequest struct {
	Path     string
	Spans []host.ByteSpan
}

// Window is the in-memory window of a vault. FocusNote looks up the note among the notes
// of the vault, and FocusFile looks up the file among its files; a path with no file in the vault
// returns ErrNotExist and is not recorded. The window returns the tabs set by SetOpenTabs.
type Window struct {
	vault *Vault

	mu         sync.Mutex
	tabs       []host.OpenTab
	activeID   string
	isAbsent   bool
	notes      []string
	focusCalls []FocusRequest
	activated  []string
}

var _ host.Window = (*Window)(nil)

// NewWindow returns the window of the vault, with no tab open.
func NewWindow(vault *Vault) *Window { return &Window{vault: vault} }

func (w *Window) hasNoWindow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isAbsent
}

func (w *Window) FocusNote(ctx context.Context, path string) (host.FocusedNote, error) {
	if w.hasNoWindow() {
		return host.FocusedNote{}, host.ErrNoWindow
	}
	found, err := w.vault.GetNotes(ctx, []string{path})
	if err != nil {
		return host.FocusedNote{}, err
	}
	ref, isNote := found[path]
	if !isNote {
		return host.FocusedNote{}, fmt.Errorf("no note at %s: %w", path, host.ErrNotExist)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.notes = append(w.notes, path)
	return host.FocusedNote{Path: ref.Path, Title: ref.Title, ID: ref.ID}, nil
}

func (w *Window) FocusFile(ctx context.Context, path string, spans []host.ByteSpan) error {
	if w.hasNoWindow() {
		return host.ErrNoWindow
	}
	_, err := w.vault.StatFile(ctx, path)
	if errors.Is(err, host.ErrNotANote) {
		return fmt.Errorf("no file at %s: %w", path, host.ErrNotExist)
	}
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.focusCalls = append(w.focusCalls, FocusRequest{Path: path, Spans: slices.Clone(spans)})
	return nil
}

func (w *Window) ReadOpenTabs(context.Context) ([]host.OpenTab, string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isAbsent {
		return nil, "", host.ErrNoWindow
	}
	return slices.Clone(w.tabs), w.activeID, nil
}

// SetOpenTabs sets the tabs the window reports from now on, and the active one.
func (w *Window) SetOpenTabs(tabs []host.OpenTab, activeID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.tabs = slices.Clone(tabs)
	w.activeID = activeID
}

// SetNoWindow makes every later call return ErrNoWindow, as a host with no window does.
func (w *Window) SetNoWindow() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isAbsent = true
}

// GetFocusedNotes returns the paths of the notes FocusNote made the current one, in order.
func (w *Window) GetFocusedNotes() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.notes)
}

// GetFocusRequests returns the FocusFile calls the window served, in order.
func (w *Window) GetFocusRequests() []FocusRequest {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.focusCalls)
}

func (w *Window) ActivateTab(_ context.Context, id string) (host.OpenTab, string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.isAbsent {
		return host.OpenTab{}, "", host.ErrNoWindow
	}
	index := slices.IndexFunc(w.tabs, func(tab host.OpenTab) bool { return tab.ID == id })
	if index < 0 {
		return host.OpenTab{}, "", fmt.Errorf("no open tab has id %q: %w", id, host.ErrNotExist)
	}
	w.activeID = id
	w.activated = append(w.activated, id)
	return w.tabs[index], id, nil
}

// GetActivatedTabs returns the ids of the tabs ActivateTab made the active one, in order.
func (w *Window) GetActivatedTabs() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.activated)
}
