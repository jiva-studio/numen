package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/plugincatalog"
	windowplugin "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/windowfocus"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/windowvaults"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func mountPlugin(w *windowWiring) {
	w.inst.SetPlugins(w.cfg.GetPluginServices(), w.cfg.GetPluginDir)
	w.inst.SetTriggers(w.cfg.plugins().OpenTriggers, w.cfg.WatchPluginChanges)
	w.inst.Mount(editor.NewRoute(windowplugin.NewRoute(w.inst.State.CountQuestions())))
}

func mountCatalog(w *windowWiring) {
	if plugins := w.cfg.getEditorCatalog(); plugins != nil {
		w.inst.Mount(editor.NewCatalogRoute(w.inst.State, plugincatalog.NewHandler(plugins)))
	}
}

// SetEditorPluginWindow gives the plugins the window: it focuses a note and the passages of a file
// in it and returns the tabs it last reported. The window reads the vault of its state on each call,
// so a vault opened later in the window is served without a new binding.
func (c Config) SetEditorPluginWindow(db *Index, opened *editor.Installation) {
	view := opened.State.GetWindow()
	focusNote := c.FocusNote(db.NoteRefs(), view)
	focusPassages := file.NewFocusPassages(c.VaultReaders(), view)
	c.SetPluginWindow(windowfocus.Window{
		GetVault: opened.State.GetCurrentVault,
		Note:     focusNote.Execute,
		Passages: func(ctx context.Context, v domain.Vault, path string, passages []domain.ByteRange) error {
			return focusPassages.Execute(ctx, v, file.FocusPassagesInput{Path: path, Passages: passages})
		},
		GetOpenTabs: opened.State.GetOpenTabs,
		ShowTab:     opened.State.ShowTab,
	})
}

// SetEditorPluginVaultList gives the plugins the list of vaults of the window and the vault it
// shows. It uses the vault use cases of the window, so a plugin and the user see the same errors.
func (c Config) SetEditorPluginVaultList(opened *editor.Installation) {
	scenarios := opened.Vaults()
	c.SetPluginVaultList(windowvaults.List{
		Add:  func(root, name string) (domain.Vault, error) { return scenarios.Add.Execute(root, name) },
		Find: func(id string) (domain.Vault, error) { return scenarios.Find.Execute(id) },
		Rename: func(ctx context.Context, v domain.Vault, name string) (domain.Vault, error) {
			return scenarios.Rename.Execute(ctx, v, name)
		},
		Remove: func(ctx context.Context, id string, current domain.VaultID, isErasing bool) error {
			return scenarios.Remove.Execute(ctx, id, current, isErasing)
		},
		GetShown:    opened.State.GetCurrentVault,
		Open:        opened.OpenPluginVault,
		CheckFolder: func(v domain.Vault) bool { return scenarios.FolderCheck.Execute(v) },
	})
}
