package editor_test

import (
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func openWindowIndex(t *testing.T, cfg testInstallation) *container.Index {
	t.Helper()
	db, err := container.OpenIndex(cfg.Config, cfg.Resources, t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func newNoteMover(db *container.Index) note.Move {
	return note.NewMove(note.MoveDeps{Readers: vaultreader.Readers{}, Writers: newVaultWriters(),
		Links: db.NoteQueries(), Names: db.NoteQueries(), Sources: db.Sources(), Now: testsupport.NewClock()})
}

func mountNoteEdit(mux *http.ServeMux, state *editor.State, procedure string, write *note.Write, create *note.Create, rename *note.Rename) {
	prose := notewindow.NewProse(state, nil, write, create, rename, state.Writes)
	_, handler := numenv1connect.NewNoteServiceHandler(notewindow.NewHandler(state.Notices, prose, nil, nil, nil))
	mux.Handle(procedure, handler)
}

func mountFileEdit(mux *http.ServeMux, state *editor.State, procedure string, move *vault.MoveFile, remove *note.RemoveFile) {
	changes := filewindow.NewChanges(state, state.Notices, move, remove, nil)
	_, handler := numenv1connect.NewFileServiceHandler(filewindow.NewHandler(nil, changes))
	mux.Handle(procedure, handler)
}
