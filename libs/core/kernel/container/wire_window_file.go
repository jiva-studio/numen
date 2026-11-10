package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func mountFile(w *windowWiring) {
	state := w.inst.State
	readers := w.cfg.VaultReaders()
	folders := filewindow.NewFolders(
		editor.NewSession(state), state.Events,
		file.NewGetFolderEntries(readers, w.db.NoteTypes()),
		file.NewGetFileKinds(readers, w.db.NoteTypes()),
		w.cfg.OpenChooseTabKind(),
		file.NewCreateFolder(folderWriters(w.cfg.vault())),
	)
	changes := filewindow.NewChanges(editor.NewSession(state), state.Writes, state.Notices, &w.vaults.Tree.Move, &w.notes.Place.Remove, &w.notes.Place.EmptyTrash)
	handler := filewindow.NewHandler(folders, changes)
	w.files = handler
	w.server = editor.NewFileServer(
		state, &w.vaults.Registry.Find, file.NewOpenFile(readers), openPluginFile(w.cfg.vault()), getMediaType(w.cfg.vault()),
	)
	w.inst.Mount(editor.NewFileRoute(state, handler))
}
