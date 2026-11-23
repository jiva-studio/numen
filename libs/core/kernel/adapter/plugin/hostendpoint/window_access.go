package hostendpoint

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// WindowAccess lets the plugins that call the host move the window: focus a note or the byte ranges of
// a file, read the open tabs and show one. It refuses each call while the window shows no vault.
type WindowAccess struct {
	current *windowsession.Runtime
	reports *windowsession.TabReports
	note    *query.FocusNote
	file    *file.FocusFile
	tabs    windowsession.ShowTab
}

// NewWindowAccess makes the access over the window the host runs in and the use cases that move it.
func NewWindowAccess(
	current *windowsession.Runtime, reports *windowsession.TabReports, note *query.FocusNote, focus *file.FocusFile, tabs windowsession.ShowTab,
) WindowAccess {
	return WindowAccess{current: current, reports: reports, note: note, file: focus, tabs: tabs}
}

func (w WindowAccess) getVault() (domain.Vault, error) {
	v := w.current.GetCurrentVault()
	if v.Path == "" {
		return domain.Vault{}, plugin.ErrNoWindow
	}
	return v, nil
}

func (w WindowAccess) FocusNote(ctx context.Context, path string) (domain.NoteRef, error) {
	v, err := w.getVault()
	if err != nil {
		return domain.NoteRef{}, err
	}
	return w.note.Execute(ctx, v, path)
}

func (w WindowAccess) FocusFile(ctx context.Context, path string, ranges []domain.ByteRange) error {
	v, err := w.getVault()
	if err != nil {
		return err
	}
	return w.file.Execute(ctx, v, file.FocusFileInput{Path: path, Ranges: ranges})
}

func (w WindowAccess) ReadOpenTabs(context.Context) (windowdomain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return windowdomain.OpenTabs{}, err
	}
	return w.reports.GetOpenTabs(), nil
}

func (w WindowAccess) ActivateTab(ctx context.Context, id string) (windowdomain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return windowdomain.OpenTabs{}, err
	}
	return w.tabs.Execute(ctx, id)
}
