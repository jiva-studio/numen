package editor

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	noteport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/notecontract"
	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
)

// pluginWindow implements PluginWindow over the window state.
type pluginWindow struct {
	state         *State
	focusNote     *note.FocusNote
	focusPassages *file.FocusPassages
}

// NewPluginWindow returns a PluginWindow over the window state: it focuses a note through the
// queries and a file of the vault through the readers, and returns the tabs the window last reported.
func NewPluginWindow(state *State, queries noteport.NoteQueries, readers vaultport.VaultReaders) pluginport.PluginWindow {
	view := state.GetWindow()
	return pluginWindow{
		state:         state,
		focusNote:     note.NewFocusNote(queries, view),
		focusPassages: file.NewFocusPassages(readers, view),
	}
}

// invalidFocusError is returned for an argument a focus use case rejects, with the use case's message.
type invalidFocusError struct{ err error }

func (e invalidFocusError) Error() string { return e.err.Error() }

func (e invalidFocusError) Unwrap() []error { return []error{e.err, plugin.ErrInvalidFocus} }

func (w pluginWindow) getVault() (domain.Vault, error) {
	v := w.state.GetCurrentVault()
	if v.Path == "" {
		return domain.Vault{}, plugin.ErrNoWindow
	}
	return v, nil
}

func (w pluginWindow) FocusNote(ctx context.Context, path string) (domain.NoteRef, error) {
	v, err := w.getVault()
	if err != nil {
		return domain.NoteRef{}, err
	}
	ref, err := w.focusNote.Execute(ctx, v, path)
	if errors.Is(err, note.ErrNoNotePath) {
		return domain.NoteRef{}, invalidFocusError{err: err}
	}
	return ref, err
}

func (w pluginWindow) FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error {
	v, err := w.getVault()
	if err != nil {
		return err
	}
	err = w.focusPassages.Execute(ctx, v, file.FocusPassagesInput{Path: path, Passages: passages})
	if errors.Is(err, file.ErrInvalidPassage) {
		return invalidFocusError{err: err}
	}
	return err
}

func (w pluginWindow) ReadOpenTabs(context.Context) (domain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return domain.OpenTabs{}, err
	}
	return w.state.GetOpenTabs(), nil
}

// defaultTabActivationBound is the wait of ActivateTab when State.TabActivationBound is zero.
const defaultTabActivationBound = 2 * time.Second

func (w pluginWindow) ActivateTab(ctx context.Context, id string) (domain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return domain.OpenTabs{}, err
	}
	report := w.state.openTabsReports.wait()
	open := w.state.GetOpenTabs()
	if !slices.ContainsFunc(open.Tabs, func(tab domain.Tab) bool { return tab.ID == id }) {
		return domain.OpenTabs{}, fmt.Errorf("editor: no open tab has id %q: %w", id, fs.ErrNotExist)
	}
	if err := w.state.GetWindow().ActivateTab(ctx, id); err != nil {
		return domain.OpenTabs{}, err
	}
	if open.ActiveID == id {
		return open, nil
	}
	wait := cmp.Or(w.state.TabActivationBound, defaultTabActivationBound)
	bound, stop := context.WithTimeout(ctx, wait)
	defer stop()
	for {
		select {
		case <-report:
			report = w.state.openTabsReports.wait()
			if open := w.state.GetOpenTabs(); open.ActiveID == id {
				return open, nil
			}
		case <-bound.Done():
			if err := ctx.Err(); err != nil {
				return domain.OpenTabs{}, err
			}
			return domain.OpenTabs{}, fmt.Errorf("editor: activate tab %q within %s: %w", id, wait, plugin.ErrTabNotShown)
		}
	}
}
