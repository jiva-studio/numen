package editor_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	filewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/file"
	notewindow "github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor/note"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

func openWindowIndex(t *testing.T, cfg testInstallation) *index.DB {
	t.Helper()
	db, err := index.Open(t.Context(), cfg.Config.IndexPath)
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return db
}

func newNoteMoverFailingToIndex(db *index.DB) note.Move {
	return note.NewMove(note.MoveDeps{Readers: vaultreader.Readers{}, Writers: newVaultWriters(),
		Links: db.NoteQueries(), Names: db.NoteQueries(), Sources: db.Sources(), Now: testsupport.NewClock(),
		Index: func(context.Context, domain.Vault, []string) error { return errors.New("a step after the move failed") }})
}

func mountNoteEdit(t *testing.T, mux *http.ServeMux, installation *editor.Installation, procedure string, write *note.Write, create *note.Create, rename *note.Rename) {
	t.Helper()
	state := editor.SessionOf(t, installation)
	prose := notewindow.NewProse(state, nil, write, create, rename, &notewindow.Writes{IsIndexOutdated: editor.PagesOf(installation).Writes.IsIndexOutdated})
	_, handler := numenv1connect.NewNoteServiceHandler(notewindow.NewHandler(state.Notices, prose, nil, nil, nil))
	mux.Handle(procedure, handler)
}

func mountFileEdit(t *testing.T, mux *http.ServeMux, installation *editor.Installation, procedure string, move *note.MoveFile, remove *note.RemoveFile) {
	t.Helper()
	state := editor.SessionOf(t, installation)
	changes := filewindow.NewChanges(state, state.Notices, move, remove, nil)
	_, handler := numenv1connect.NewFileServiceHandler(filewindow.NewHandler(nil, changes))
	mux.Handle(procedure, handler)
}
