package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	windowplugin "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(w.cfg.GetPluginServices(), w.cfg.GetPluginDir)
	w.inst.Mount(editor.NewRoute(windowplugin.NewRoute(w.inst.State.Gate.CountQuestions())))
}

// SetEditorPluginWindow gives the plugins the window: it focuses a note and the passages of a file
// in it and returns the tabs it last reported. The window reads the vault of its state on each call,
// so a vault opened later in the window is served without a new binding.
func (c Config) SetEditorPluginWindow(db *vault.Index, opened *editor.Installation) {
	view := GetWindowView(opened)
	focusPassages := file.NewFocusPassages(c.VaultReaders(), view)
	c.SetPluginWindow(windowsession.NewPluginWindow(
		&opened.State.Runtime,
		c.FocusNote(db.NoteQueries.NoteRefs, view),
		windowsession.PassagesFocusFunc(func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error {
			return focusPassages.Execute(ctx, v, file.FocusPassagesInput{Path: path, Passages: passages})
		}),
		opened.ShowTabs(),
	))
}

// SetEditorPluginVaultList gives the plugins the list of vaults of the window and the vault it
// shows. It uses the vault use cases of the window, so a plugin and the user see the same errors.
func (c Config) SetEditorPluginVaultList(opened *editor.Installation) {
	scenarios := opened.Vaults()
	c.SetPluginVaultList(windowsession.NewPluginVaults(&opened.State.Runtime, windowsession.VaultOps{
		Edit:    windowsession.VaultEdits{Add: scenarios.Change.Add, Rename: scenarios.Change.Rename, Remove: scenarios.Change.Remove},
		Find:    scenarios.Registry.Find,
		Missing: scenarios.Registry.FolderCheck,
		Open:    opened,
	}))
}
