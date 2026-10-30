package indexing

import (
	"context"
	"errors"
	"hash/fnv"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

var before = []string{"river", "mountain", "forest", "meadow", "harbour", "valley", "island", "desert"}

// noteWith is a note of n words of one vocabulary, long enough to be cut into
// windows that carry a vector.
func noteWith(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return "---\ntitle: Note\n---\n\n" + strings.Join(out, " ") + "\n"
}

// recorder keeps the list of what is being done, as the window shows it.
type recorder struct {
	tasks *run.Tasks
}

func newRecorder() *recorder { return &recorder{tasks: run.NewTasks()} }

func (r *recorder) ReportTask(pos run.Task)                        { r.tasks.Set(pos) }
func (r *recorder) FinishTask(id string)                           { r.tasks.Remove(id) }
func (r *recorder) ReportError(error)                              {}
func (r *recorder) ReportNotes(domain.Vault, int)                  {}
func (r *recorder) ReportTexts(domain.Vault, filetext.IndexResult) {}

// findTask is one piece of work as the list holds it, or nothing where the list
// does not hold it.
func findTask(t *testing.T, r *recorder, id string) *run.Task {
	t.Helper()
	for _, item := range r.tasks.List() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

// reading is an index and the assembly over it, for a pass driven without a
// window in front of it.
func reading(t *testing.T) (container.EditorAssembly, *container.Index) {
	t.Helper()

	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	testsupport.CloseAfter(t, db)
	return container.NewEditorAssembly(cfg, nil, db), db
}

// newPass is the indexing of one vault over an embedder, reporting into events.
func newPass(
	made container.EditorAssembly, db *container.Index, events Events, embedder embeddingport.Embedder, recipe string,
	v domain.Vault,
) *pass {
	worker := NewWorker(Config{
		Reading:  made.ReadWholeVault,
		Embedder: embedder,
		VectorCounts: func(ctx context.Context, v domain.Vault) (int64, int64, error) {
			return db.Progress().GetProgress(ctx, v.ID, recipe)
		},
		Events: events,
		Quiet:  time.Hour,
		After:  time.After,
	})
	return &pass{worker: worker, session: worker.NewSession(), v: v}
}

// cut reads the vault's notes into the index as chunks with pending vectors.
func cut(t *testing.T, made container.EditorAssembly, db *container.Index, v domain.Vault) {
	t.Helper()

	scan := vault.NewScan(
		made.GetVaultReaders(),
		db.Vaults(),
		db.Notes(),
		db.Queries(),
		db.Maintenance(),
	)
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
}

// fit prepares the index to hold the vectors of a model.
func fit(t *testing.T, db *container.Index, model embeddingport.Embedder) {
	t.Helper()
	if err := db.FitVectors(t.Context(), model.GetModel().Dimensions, model.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}
}

// fakeEmbedder is a model that returns a direction derived from the input text.
// It makes no calls outside this process.
type fakeEmbedder struct {
	dims int
}

func (a *fakeEmbedder) GetModel() embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{Name: "fake", Dimensions: a.dims}
}

func (a *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		out = append(out, direction(text, a.dims))
	}
	return out, nil
}

func (*fakeEmbedder) Close() error { return nil }

func (*fakeEmbedder) Concurrency() int { return 1 }

// direction is a vector that belongs to one text and no other.
func direction(text string, dims int) []float32 {
	sum := fnv.New64a()
	sum.Write([]byte(text))
	numbers := rand.New(rand.NewPCG(sum.Sum64(), 0x9E3779B97F4A7C15))

	out := make([]float32, dims)
	for i := range out {
		out[i] = float32(numbers.NormFloat64())
	}
	return out
}

// sulking is a model that turns down every asking.
type sulking struct {
	*fakeEmbedder
}

func (*sulking) Embed(context.Context, []string) ([][]float32, error) {
	return nil, errors.New("the model is not answering")
}

func (*sulking) Concurrency() int { return 1 }

// A model that is there and not answering leaves the pass in the list under the
func TestAPassThatCouldNotEmbedStaysInTheList(t *testing.T) {
	model := &sulking{fakeEmbedder: &fakeEmbedder{dims: 64}}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	pos := findTask(t, events, MakingVectors)
	if pos == nil {
		t.Fatal("the pass that could not embed took itself out of the list")
	}
	if pos.Error == "" {
		t.Errorf("the pass is shown as running: %+v", *pos)
	}
}

// A completed pass does not leave its entry in the task list.
func TestAPassThatEmbeddedPreservesTheList(t *testing.T) {
	model := &fakeEmbedder{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, MakingVectors); pos != nil {
		t.Errorf("a pass that embedded what was unembedded is still being done: %+v", *pos)
	}
}

// peeking is a model that records the task list as it was when the first
// vector was requested; the status corner displays that list.
type peeking struct {
	*fakeEmbedder
	tasks *run.Tasks

	mu    sync.Mutex
	while []run.Task
}

func (p *peeking) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	p.mu.Lock()
	if p.while == nil {
		p.while = p.tasks.List()
	}
	p.mu.Unlock()
	return p.fakeEmbedder.Embed(ctx, texts)
}

func (*peeking) Concurrency() int { return 1 }

// opening returns the task as it was when the first vector was requested.
func (p *peeking) opening(id string) (run.Task, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, item := range p.while {
		if item.ID == id {
			return item, true
		}
	}
	return run.Task{}, false
}

// Indexing is always of something, and a row carrying the word alone leaves a
// person asking which of their notes it is on.
func TestIndexingNamesTheSourceItIsOn(t *testing.T) {
	events := newRecorder()
	watching := &peeking{fakeEmbedder: &fakeEmbedder{dims: 64}, tasks: events.tasks}
	made, db := reading(t)
	fit(t, db, watching)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	newPass(made, db, events, watching, watching.GetModel().Recipe(), v).embed(t.Context())

	pos, target := watching.opening(MakingVectors)
	if !target {
		t.Fatal("nothing was being indexed while a vector was asked for")
	}
	if pos.About != "Note.md" {
		t.Errorf("the pass says it is indexing %q", pos.About)
	}
}

// A library that would not be read is a vault searching less than it holds, and
// the pass is left standing under what stopped it.
func TestBooksThatCouldNotBeReadStayInTheList(t *testing.T) {
	made, db := reading(t)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	events := newRecorder()
	// The index queried for pending vectors is closed, so the query fails.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	newPass(made, db, events, nil, "", v).indexTexts(t.Context(), false)

	pos := findTask(t, events, IndexingTexts)
	if pos == nil {
		t.Fatal("the pass that could not read the books took itself out of the list")
	}
	if pos.Error == "" {
		t.Errorf("the pass is shown as running: %+v", *pos)
	}
}

// overheard is a model that records the task list at every call, as the
// status corner displays it.
type overheard struct {
	*fakeEmbedder
	tasks *run.Tasks

	mu      sync.Mutex
	visited [][]run.Task
}

// GetModel is called at the start of every pass, before a source is opened.
func (o *overheard) GetModel() embeddingport.EmbeddingModel {
	o.keep()
	return o.fakeEmbedder.GetModel()
}

func (o *overheard) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	o.keep()
	return o.fakeEmbedder.Embed(ctx, texts)
}

func (*overheard) Concurrency() int { return 1 }

func (o *overheard) keep() {
	o.mu.Lock()
	o.visited = append(o.visited, o.tasks.List())
	o.mu.Unlock()
}

func (o *overheard) lists() [][]run.Task {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.visited
}

// The status row must name its current work. A pass that enters the task list
// before opening a source must not show the bare word "Indexing" over an empty
// line at zero progress while the first vectors are computed.
func TestIndexingIsNeverAWordWithNothingBelowIt(t *testing.T) {
	events := newRecorder()
	model := &fakeEmbedder{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	overVal := &overheard{fakeEmbedder: model, tasks: events.tasks}

	newPass(made, db, events, overVal, model.GetModel().Recipe(), v).embed(t.Context())

	for _, list := range overVal.lists() {
		for _, item := range list {
			if item.ID != MakingVectors || item.Error != "" {
				continue
			}
			if item.About == "" {
				t.Errorf("the pass stood in the list saying nothing but %q", item.Title)
			}
			if item.Total > 0 && item.Count == 0 {
				t.Errorf("the pass drew a share of nothing: %+v", item)
			}
		}
	}
}

// nudging is a model that has a recognition write a batch of pages while the
// vector pass is running, and holds the pass until the wake has stopped it.
type nudging struct {
	*fakeEmbedder
	wake chan struct{}

	once sync.Once
}

func (n *nudging) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	isStopped := false
	n.once.Do(func() {
		signal(n.wake)
		<-ctx.Done()
		isStopped = true
	})
	if isStopped {
		return nil, ctx.Err()
	}
	return n.fakeEmbedder.Embed(ctx, texts)
}

func (*nudging) Concurrency() int { return 1 }

// A recognition writes a batch of pages every few minutes, and those pages
// are cut on the goroutine of the vector pass. A vault with a hundred thousand
// pending vectors needs hours to embed them.
func TestARecognitionStopsTheVectorPass(t *testing.T) {
	events := newRecorder()
	model := &nudging{fakeEmbedder: &fakeEmbedder{dims: 64}, wake: make(chan struct{}, 1)}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.read = model.wake
	p.embed(t.Context())

	target, embedded, err := db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target == 0 {
		t.Fatal("the vault unembedded no vector, and the pass had nothing to stand aside from")
	}
	if embedded == target {
		t.Error("the pass embedded the whole vault while a recognition waited to be cut")
	}

	// The wake is left where it was found: the loop reads it next and cuts what
	// the recognition wrote.
	select {
	case <-model.wake:
	default:
		t.Error("the wake a recognition raised was taken and not put back")
	}

	if pos := findTask(t, events, MakingVectors); pos != nil && pos.Error != "" {
		t.Errorf("standing aside is shown as a failure: %q", pos.Error)
	}

	// After resuming, the pass queries the index for the vectors still pending.
	again := newPass(made, db, events, model.fakeEmbedder, model.GetModel().Recipe(), v)
	again.embed(t.Context())

	target, embedded, err = db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if embedded != target {
		t.Errorf("the pass taken up again left %d of %d chunks without a vector", target-embedded, target)
	}
}

// A change of the embedding model wakes the sources while a pass embeds under
// the previous model. The pass stops before sending batches that the new model
// embeds at another width.
func TestAChangeOfSourcesStopsTheVectorPass(t *testing.T) {
	events := newRecorder()
	model := &nudging{fakeEmbedder: &fakeEmbedder{dims: 64}, wake: make(chan struct{}, 1)}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.sources = model.wake
	p.embed(t.Context())

	target, embedded, err := db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target == 0 {
		t.Fatal("the vault had no pending vector, and the pass had nothing to stop")
	}
	if embedded == target {
		t.Error("the pass embedded the whole vault while the sources were woken")
	}

	select {
	case <-model.wake:
	default:
		t.Error("the wake of the sources was taken and not put back")
	}
}

// The pass for a note written waits for the quiet period, and a second write
// puts it off again.
func TestAWakeForNotesIsLeftOnceWhateverTheNumberOfChanges(t *testing.T) {
	worker := NewWorker(Config{})
	for range 3 {
		worker.WakeNotes()
	}
	if got := len(worker.notes); got != 1 {
		t.Errorf("%d wakes are waiting, want 1", got)
	}
}

func TestVersionChangeRequestsExtractionOfTheFileAgain(t *testing.T) {
	worker := NewWorker(Config{Quiet: time.Hour, After: time.After})
	session := worker.NewSession()
	v := domain.Vault{ID: "v1"}

	session.ReportVersionsChanged(v, "books/scan.pdf")
	session.ReportVersionsChanged(v, "books/scan.pdf")
	session.ReportVersionsChanged(v, "talks/a.mp3")

	select {
	case <-worker.read:
	default:
		t.Fatal("a change of versions woke nothing")
	}
	got := session.drain()
	if len(got) != 2 || got["books/scan.pdf"].ID != "v1" || got["talks/a.mp3"].ID != "v1" {
		t.Fatalf("waiting to be cut: %v, want each changed file once", got)
	}
}

// A change to a file's versions starts a pass that cuts the file again and
// embeds the vault's pending vectors.
func TestVersionChange_EmbedsPendingVaultContent(t *testing.T) {
	model := &fakeEmbedder{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	p := newPass(made, db, newRecorder(), model, model.GetModel().Recipe(), v)
	p.session.ReportVersionsChanged(v, "books/scan.pdf")
	<-p.worker.read
	p.embedRecognised(t.Context())

	target, embedded, err := db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target == 0 || embedded != target {
		t.Errorf("after a version change %d of %d chunks carry a vector", embedded, target)
	}
}

// A note is cut from its own words, so a change to the versions of a note
// keeps the chunks the note was cut into.
func TestVersionChangeOfANoteKeepsItsChunks(t *testing.T) {
	model := &fakeEmbedder{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	unembedded, _, err := db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if unembedded == 0 {
		t.Fatal("the note was cut into no chunk that needs a vector")
	}

	p := newPass(made, db, newRecorder(), model, model.GetModel().Recipe(), v)
	p.session.ReportVersionsChanged(v, "Note.md")
	<-p.worker.read
	p.embedRecognised(t.Context())

	target, embedded, err := db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target != unembedded || embedded != target {
		t.Errorf("after a version change of the note %d of %d chunks carry a vector, want %d of %d",
			embedded, target, unembedded, unembedded)
	}
}
