// Package windowfocus lets the plugins that call the host move the window: focus a note or the
// passages of a file, read the open tabs and show one. The window supplies each operation; the
// package refuses them while the window shows no vault.
package windowfocus

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Window is the main window of the vault, as the window moves it.
type Window struct {
	GetVault    func() domain.Vault
	Note        func(ctx context.Context, v domain.Vault, path string) (domain.NoteRef, error)
	Passages    func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error
	GetOpenTabs func() domain.OpenTabs
	ShowTab     func(ctx context.Context, id string) (domain.OpenTabs, error)
}

func (w Window) getVault() (domain.Vault, error) {
	v := w.GetVault()
	if v.Path == "" {
		return domain.Vault{}, plugin.ErrNoWindow
	}
	return v, nil
}

// FocusNote makes the note at the path the current one and returns it.
func (w Window) FocusNote(ctx context.Context, path string) (domain.NoteRef, error) {
	v, err := w.getVault()
	if err != nil {
		return domain.NoteRef{}, err
	}
	return w.Note(ctx, v, path)
}

// FocusPassages opens the file at the path at the first passage and highlights every passage.
func (w Window) FocusPassages(ctx context.Context, path string, passages []domain.ByteRange) error {
	v, err := w.getVault()
	if err != nil {
		return err
	}
	return w.Passages(ctx, v, path, passages)
}

// ReadOpenTabs returns the tabs the window last reported, and the active one.
func (w Window) ReadOpenTabs(context.Context) (domain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return domain.OpenTabs{}, err
	}
	return w.GetOpenTabs(), nil
}

// ActivateTab makes the open tab with the id the active tab, and returns the open tabs as the window
// reports them afterwards.
func (w Window) ActivateTab(ctx context.Context, id string) (domain.OpenTabs, error) {
	if _, err := w.getVault(); err != nil {
		return domain.OpenTabs{}, err
	}
	return w.ShowTab(ctx, id)
}
