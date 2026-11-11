package windowsession

import (
	"context"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// NoteFocus makes a note of a vault the current one.
type NoteFocus interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteRef, error)
}

// PassagesFocus opens a file of a vault at its passages.
type PassagesFocus interface {
	FocusPassages(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error
}

// PassagesFocusFunc is PassagesFocus as a function.
type PassagesFocusFunc func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error

// FocusPassages calls f.
func (f PassagesFocusFunc) FocusPassages(
	ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange,
) error {
	return f(ctx, v, path, passages)
}

// TabShowing makes an open tab the active one.
type TabShowing interface {
	Execute(ctx context.Context, id string) (windowdomain.OpenTabs, error)
}

// PluginWindow lets the plugins that call the host move the window: focus a note or the passages of
// a file, read the open tabs and show one. It refuses each call while the window shows no vault.
type PluginWindow struct {
	current  currentVault
	reports  *TabReports
	note     NoteFocus
	passages PassagesFocus
	tabs     TabShowing
}

// NewPluginWindow makes the window plugins move, over the runtime and the tab reports of the window.
func NewPluginWindow(rt *Runtime, reports *TabReports, note NoteFocus, passages PassagesFocus, tabs TabShowing) PluginWindow {
	return PluginWindow{current: rt, reports: reports, note: note, passages: passages, tabs: tabs}
}

func (w PluginWindow) getVault() (domain.Vault, error) {
	v := w.current.GetCurrentVault()
	if v.Path == "" {
		return domain.Vault{}, plugin.ErrNoWindow
	}
	return v, nil
}

// FocusNote makes the note at the path the current one and returns it.
func (w PluginWindow) FocusNote(ctx context.Context, path string) (domain.NoteRef, error) {
	v, err := w.getVault()
	if err != nil {
		return domain.NoteRef{}, err
	}
	return w.note.Execute(ctx, v, path)
}

// FocusPassages opens the file at the path at the first passage and highlights every passage.
func (w PluginWindow) FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error {
	v, err := w.getVault()
	if err != nil {
		return err
	}
	return w.passages.FocusPassages(ctx, v, path, passages)
}

// ReadOpenTabs returns the tabs the window last reported, and the active one.
func (w PluginWindow) ReadOpenTabs(context.Context) (windowdomain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return windowdomain.OpenTabs{}, err
	}
	return w.reports.GetOpenTabs(), nil
}

// ActivateTab makes the open tab with the id the active tab, and returns the open tabs as the window
// reports them afterwards.
func (w PluginWindow) ActivateTab(ctx context.Context, id string) (windowdomain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return windowdomain.OpenTabs{}, err
	}
	return w.tabs.Execute(ctx, id)
}
