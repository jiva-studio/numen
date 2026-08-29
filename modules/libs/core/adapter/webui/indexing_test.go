package webui

import (
	"io"
	"math"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// TestAPassThatCouldNotEmbedStaysInTheList. A window opened from a desktop
// entry has no terminal, so the one pass that quietly costs a person a
// searchable note says so where they are and stays said.
func TestAPassThatCouldNotEmbedStaysInTheList(t *testing.T) {
	// Every asking is turned down, which is what a model that is there and not
	// answering looks like from here.
	model := sulks(64, math.MaxInt)
	cfg, db := reading(t)
	if err := db.FitVectors(t.Context(), model.Model().Dimensions, model.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{
		Tasking:  task.New(),
		Progress: db.Progress(),
	}
	api.show(v)
	api.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	embedSources(t.Context(), cfg, db, api, v, filesystem.Readers{}, model)

	at := listed(t, api, makingVectors)
	if at == nil {
		t.Fatal("the pass that could not embed took itself out of the list")
	}
	if at.Failed == "" {
		t.Errorf("the pass is shown as running: %+v", *at)
	}
}

// TestAPassThatEmbeddedLeavesTheList. What ended is not what is being done, and
// a list a finished pass stays in is a list nobody reads.
func TestAPassThatEmbeddedLeavesTheList(t *testing.T) {
	model := &asked{dims: 64}
	cfg, db := reading(t)
	if err := db.FitVectors(t.Context(), model.Model().Dimensions, model.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{
		Tasking:  task.New(),
		Progress: db.Progress(),
	}
	api.show(v)
	api.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	embedSources(t.Context(), cfg, db, api, v, filesystem.Readers{}, model)

	if at := listed(t, api, makingVectors); at != nil {
		t.Errorf("a pass that embedded what was owed is still being done: %+v", *at)
	}
}

// TestBooksThatCouldNotBeReadStayInTheList. A library that would not be read is
// a vault searching less than it holds, and the pass is left standing under
// what stopped it.
func TestBooksThatCouldNotBeReadStayInTheList(t *testing.T) {
	cfg, db := reading(t)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Tasking: task.New()}
	api.show(v)
	// The index the pass asks what owes its text is gone, which is what it
	// looks like from here when the answer cannot be had.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	readSources(t.Context(), cfg, db, api, v, filesystem.Readers{}, nil, io.Discard)

	at := listed(t, api, readingBooks)
	if at == nil {
		t.Fatal("the pass that could not read the books took itself out of the list")
	}
	if at.Failed == "" {
		t.Errorf("the pass is shown as running: %+v", *at)
	}
}

// reading is an index and the settings it was opened from, for a pass driven
// without a window in front of it.
func reading(t *testing.T) (container.Config, *container.Index) {
	t.Helper()

	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return cfg, db
}

// cut reads the vault's notes, so that its chunks are in the index owing their
// vectors.
func cut(t *testing.T, db *container.Index, api *API) {
	t.Helper()

	scan := usecase.Scan{
		Readers:     filesystem.Readers{},
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
	if _, err := scan.Execute(t.Context(), api.Showing()); err != nil {
		t.Fatal(err)
	}
}

// listed is one piece of work as the list holds it, or nothing where the list
// does not hold it.
func listed(t *testing.T, api *API, id string) *task.Task {
	t.Helper()

	for _, at := range api.Tasking.List() {
		if at.ID == id {
			return &at
		}
	}
	return nil
}
