package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func mountFile(w *windowWiring) {
	state := w.inst.State
	readers := w.cfg.VaultReaders()
	handler := editor.NewFileHandler(state)
	handler.SetFolders(
		file.NewGetFolderEntries(readers, w.db.NoteTypes()),
		file.NewGetFileKinds(readers, w.db.NoteTypes()),
		file.NewCreateFolder(w.cfg.vault().FolderWriters()),
	)
	handler.SetTabKindChoice(w.cfg.OpenChooseTabKind())
	handler.SetScenarios(w.notes, w.vaults)
	w.files = handler
	w.server = editor.NewFileServer(
		state, &w.vaults.Find, file.NewOpenFile(readers), w.cfg.vault().OpenPluginFile(), w.cfg.vault().GetMediaType(),
	)
	w.inst.Mount(editor.NewFileRoute(state, handler))
}
