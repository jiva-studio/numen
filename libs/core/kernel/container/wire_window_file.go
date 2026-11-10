package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

// mountFile mounts the service that lists, makes, moves and removes the files of the vault, and
// makes the server that reads a file over the loopback socket.
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
