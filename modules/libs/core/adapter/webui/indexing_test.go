package webui

import (
	"context"
	"io"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, model)

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
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, model)

	if at := listed(t, api, makingVectors); at != nil {
		t.Errorf("a pass that embedded what was owed is still being done: %+v", *at)
	}
}

// TestIndexingNamesTheSourceItIsOn. Indexing is always of something, and a row
// carrying the word alone leaves a person asking which of their notes it is on.
func TestIndexingNamesTheSourceItIsOn(t *testing.T) {
	model := &asked{dims: 64}
	watching := &peeking{asked: model}
	cfg, db := reading(t)
	if err := db.FitVectors(t.Context(), model.Model().Dimensions, model.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)
	watching.tasks = api.Window.Tasking

	embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, watching)

	at, held := watching.opening(makingVectors)
	if !held {
		t.Fatal("nothing was being indexed while a vector was asked for")
	}
	if at.About != "Note.md" {
		t.Errorf("the pass says it is indexing %q", at.About)
	}
}

// TestAVaultOwingNoVectorWaitsForNoModel. Every path into this pass runs on the
// one goroutine that also reads the books and cuts what a recognition wrote, and
// a model is fetched and compiled in minutes. A vault that owes nothing holds
// that goroutine for nothing.
func TestAVaultOwingNoVectorWaitsForNoModel(t *testing.T) {
	model := &asked{dims: 64}
	cfg, db := reading(t)

	// The notes are on disk and nothing has cut them, so the index holds no
	// chunk and owes no vector.
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())

	// The weights are still coming down, and in this test they never land.
	arriving := embedding.Arriving(model.Model())

	over := make(chan struct{})
	go func() {
		defer close(over)
		embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, arriving.Filling())
	}()

	select {
	case <-over:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass is waiting for a model to embed nothing with")
	}
}

// TestNothingIsIndexedWhileTheModelIsOnItsWay. Fetching a model is not
// indexing. A row that calls it by the name of the work that follows sits at no
// share of nothing while the weights come down.
func TestNothingIsIndexedWhileTheModelIsOnItsWay(t *testing.T) {
	model := &asked{dims: 64}
	cfg, db := reading(t)
	if err := db.FitVectors(t.Context(), model.Model().Dimensions, model.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	arriving := embedding.Arriving(model.Model())
	over := make(chan struct{})
	go func() {
		defer close(over)
		embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, arriving.Filling())
	}()

	for range 20 {
		if at := listed(t, api, makingVectors); at != nil {
			t.Fatalf("a model still on its way is shown as indexing: %+v", *at)
		}
		time.Sleep(time.Millisecond)
	}

	arriving.Landed(model, nil)
	<-over

	if at := listed(t, api, makingVectors); at != nil {
		t.Errorf("the pass that embedded what was owed is still being done: %+v", *at)
	}
	if model.times() == 0 {
		t.Error("the model landed and nothing was embedded")
	}
}

// peeking is a model that says what was being done when it was asked for a
// vector, which is what a person watching the corner would have read.
type peeking struct {
	*asked
	tasks *task.Tasks

	mu    sync.Mutex
	while []task.Task
}

func (p *peeking) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	p.mu.Lock()
	if p.while == nil {
		p.while = p.tasks.List()
	}
	p.mu.Unlock()
	return p.asked.Embed(ctx, texts)
}

// opening is one piece of work as it stood when the first vector was asked for.
func (p *peeking) opening(id string) (task.Task, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, at := range p.while {
		if at.ID == id {
			return at, true
		}
	}
	return task.Task{}, false
}

// TestBooksThatCouldNotBeReadStayInTheList. A library that would not be read is
// a vault searching less than it holds, and the pass is left standing under
// what stopped it.
func TestBooksThatCouldNotBeReadStayInTheList(t *testing.T) {
	cfg, db := reading(t)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.show(v)
	// The index the pass asks what owes its text is gone, which is what it
	// looks like from here when the answer cannot be had.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	readSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, nil, io.Discard)

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
		Readers:     filesystem.VaultReaders{},
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

	for _, at := range api.Window.Tasking.List() {
		if at.ID == id {
			return &at
		}
	}
	return nil
}

// TestIndexingIsNeverAWordWithNothingUnderIt. The row a person watches for
// minutes must say what it is on. A pass that enters the list before it has
// opened a source stands there as the word "Indexing" over an empty line, at a
// share of nothing, for as long as the first vectors take to come back.
func TestIndexingIsNeverAWordWithNothingUnderIt(t *testing.T) {
	model := &asked{dims: 64}
	cfg, db := reading(t)
	if err := db.FitVectors(t.Context(), model.Model().Dimensions, model.Model().Recipe()); err != nil {
		t.Fatal(err)
	}

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	api := &API{Window: &wire.Window{Named: wire.Editor, Tasking: task.New()}}
	api.Indexing.Progress = db.Progress()
	api.show(v)
	api.Indexing.Recipe.Store(model.Model().Recipe())
	cut(t, db, api)

	// A provider that answers over a network is here the moment it is made, so
	// there is no arrival to wait for and the pass begins at once.
	over := &overheard{asked: model, tasks: api.Window.Tasking}
	held := embedding.Arriving(model.Model())
	held.Landed(over, nil)

	embedSources(t.Context(), cfg, db, api, v, filesystem.VaultReaders{}, held.Filling())

	for _, list := range over.lists() {
		for _, at := range list {
			if at.ID != makingVectors || at.Failed != "" {
				continue
			}
			if at.About == "" {
				t.Errorf("the pass stood in the list saying nothing but %q", at.Doing)
			}
			if at.Total > 0 && at.Count == 0 {
				t.Errorf("the pass drew a share of nothing: %+v", at)
			}
		}
	}
}

// overheard is a model that keeps the list of what is being done as it stood
// every time it was asked anything, which is what a person watching the corner
// would have read.
type overheard struct {
	*asked
	tasks *task.Tasks

	mu   sync.Mutex
	seen [][]task.Task
}

// Model is asked at the head of every pass, before a source has been opened.
func (o *overheard) Model() port.EmbeddingModel {
	o.keep()
	return o.asked.Model()
}

func (o *overheard) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	o.keep()
	return o.asked.Embed(ctx, texts)
}

func (o *overheard) keep() {
	o.mu.Lock()
	o.seen = append(o.seen, o.tasks.List())
	o.mu.Unlock()
}

func (o *overheard) lists() [][]task.Task {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.seen
}
