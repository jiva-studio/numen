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

// FileFocus opens a file of a vault at its byte ranges.
type FileFocus interface {
	FocusFile(ctx context.Context, v domain.Vault, path string, ranges []domain.ByteRange) error
}

type FileFocusFunc func(ctx context.Context, v domain.Vault, path string, ranges []domain.ByteRange) error

// FocusFile calls f.
func (f FileFocusFunc) FocusFile(
	ctx context.Context, v domain.Vault, path string, ranges []domain.ByteRange,
) error {
	return f(ctx, v, path, ranges)
}

// TabShowing makes an open tab the active one.
type TabShowing interface {
	Execute(ctx context.Context, id string) (windowdomain.OpenTabs, error)
}

// PluginWindow lets the plugins that call the host move the window: focus a note or the byte ranges of
// a file, read the open tabs and show one. It refuses each call while the window shows no vault.
type PluginWindow struct {
	current currentVault
	reports *TabReports
	note    NoteFocus
	files   FileFocus
	tabs    TabShowing
}

func NewPluginWindow(rt *Runtime, reports *TabReports, note NoteFocus, files FileFocus, tabs TabShowing) PluginWindow {
	return PluginWindow{current: rt, reports: reports, note: note, files: files, tabs: tabs}
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

// FocusFile opens the file at the path at the first range and highlights every range.
func (w PluginWindow) FocusFile(ctx context.Context, path string, ranges []domain.ByteRange) error {
	v, err := w.getVault()
	if err != nil {
		return err
	}
	return w.files.FocusFile(ctx, v, path, ranges)
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
