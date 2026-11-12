package indexing

import (
	"context"
	"errors"
	"hash/fnv"
	"io"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/indexport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

func getBeforeNames() []string {
	return []string{"river", "mountain", "forest", "meadow", "harbor", "valley", "island", "desert"}
}

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

func (r *recorder) ReportTask(pos work.Task)                           { r.tasks.Set(pos) }
func (r *recorder) FinishTask(id string)                               { r.tasks.Remove(id) }
func (r *recorder) ReportError(error)                                  {}
func (r *recorder) ReportNotes(domain.Vault, int)                      {}
func (r *recorder) ReportTexts(domain.Vault, vaultreading.IndexResult) {}

func findTask(t *testing.T, r *recorder, id string) *work.Task {
	t.Helper()
	for _, item := range r.tasks.List() {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

func newPass(
	made reading, db *WorkerIndex, events Events, embedder embeddingport.Embedder, recipe string,
	v domain.Vault,
) *testPass {
	worker := NewWorker(Config{
		Reading:  made.ReadWholeVault,
		Embedder: embedder,
		VectorCounts: func(ctx context.Context, v domain.Vault) (int64, int64, error) {
			return db.Progress.GetProgress(ctx, v.ID, recipe)
		},
		Events: events,
		Quiet:  time.Hour,
		After:  time.After,
	})
	return &testPass{pass: worker.newPass(worker.NewSession(), v, false), worker: worker}
}

func cut(t *testing.T, made reading, db *WorkerIndex, v domain.Vault) {
	t.Helper()

	scan := vault.NewScan(
		made.GetVaultReaders(),
		db.Vaults,
		db.Notes,
		db.FileStats,
		db.Maintenance, nil,
	)
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
}

func fit(t *testing.T, db *WorkerIndex, model embeddingport.Embedder) {
	t.Helper()
	if err := db.Vectors.FitVectors(t.Context(), model.GetModel().Dimensions, model.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}
}

// fakeEmbedder is a model that returns a direction derived from the input text.
// It makes no calls outside this process.
type fakeEmbedder struct {
	dims int
}

func (a *fakeEmbedder) GetModel() embedding.Model {
	return embedding.Model{Name: "fake", Dimensions: a.dims}
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

func testAPassThatCouldNotEmbedStaysInTheList(t *testing.T, openReading openFunc) {
	t.Helper()
	model := &sulking{fakeEmbedder: &fakeEmbedder{dims: 64}}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	pos := findTask(t, events, work.MakingVectors)
	if pos == nil {
		t.Fatal("the pass that could not embed took itself out of the list")
	}
	if pos.Error == "" {
		t.Errorf("the pass is shown as running: %+v", *pos)
	}
}

func testAPassThatEmbeddedPreservesTheList(t *testing.T, openReading openFunc) {
	t.Helper()
	model := &fakeEmbedder{dims: 64}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, work.MakingVectors); pos != nil {
		t.Errorf("a pass that embedded what was unembedded is still being done: %+v", *pos)
	}
}

// peeking is a model that records the task list as it was when the first
// vector was requested; the status corner displays that list.
type peeking struct {
	*fakeEmbedder
	tasks *run.Tasks

	mu    sync.Mutex
	while []work.Task
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

func (p *peeking) opening(id string) (work.Task, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, item := range p.while {
		if item.ID == id {
			return item, true
		}
	}
	return work.Task{}, false
}

func testIndexingNamesTheSourceItIsOn(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	watching := &peeking{fakeEmbedder: &fakeEmbedder{dims: 64}, tasks: events.tasks}
	made, db := openReading(t)
	fit(t, db, watching)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	newPass(made, db, events, watching, watching.GetModel().Recipe(), v).embed(t.Context())

	pos, target := watching.opening(work.MakingVectors)
	if !target {
		t.Fatal("nothing was being indexed while a vector was asked for")
	}
	if pos.About != "Note.md" {
		t.Errorf("the pass says it is indexing %q", pos.About)
	}
}

func testBooksThatCouldNotBeReadStayInTheList(t *testing.T, openReading openFunc) {
	t.Helper()
	made, db := openReading(t)
	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	events := newRecorder()
	// The index queried for pending vectors is closed, so the query fails.
	if err := db.Closer.Close(); err != nil {
		t.Fatal(err)
	}

	newPass(made, db, events, nil, "", v).indexTexts(t.Context(), false)

	pos := findTask(t, events, work.IndexingTexts)
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
	visited [][]work.Task
}

// GetModel is called at the start of every pass, before a source is opened.
func (o *overheard) GetModel() embedding.Model {
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

func (o *overheard) lists() [][]work.Task {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.visited
}

func testIndexingIsNeverAWordWithNothingBelowIt(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &fakeEmbedder{dims: 64}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	overVal := &overheard{fakeEmbedder: model, tasks: events.tasks}

	newPass(made, db, events, overVal, model.GetModel().Recipe(), v).embed(t.Context())

	for _, list := range overVal.lists() {
		for _, item := range list {
			if item.ID != work.MakingVectors || item.Error != "" {
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

func testARecognitionStopsTheVectorPass(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &nudging{fakeEmbedder: &fakeEmbedder{dims: 64}, wake: make(chan struct{}, 1)}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.wakes.read = model.wake
	p.embed(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
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

	if pos := findTask(t, events, work.MakingVectors); pos != nil && pos.Error != "" {
		t.Errorf("standing aside is shown as a failure: %q", pos.Error)
	}

	// After resuming, the pass queries the index for the vectors still pending.
	again := newPass(made, db, events, model.fakeEmbedder, model.GetModel().Recipe(), v)
	again.embed(t.Context())

	target, embedded, err = db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if embedded != target {
		t.Errorf("the pass taken up again left %d of %d chunks without a vector", target-embedded, target)
	}
}

func testAChangeOfSourcesStopsTheVectorPass(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &nudging{fakeEmbedder: &fakeEmbedder{dims: 64}, wake: make(chan struct{}, 1)}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.wakes.sources = model.wake
	p.embed(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
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

func testANoteSaveStopsTheVectorPass(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &nudging{fakeEmbedder: &fakeEmbedder{dims: 64}, wake: make(chan struct{}, 1)}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, events, model, model.GetModel().Recipe(), v)
	p.worker.wakes.notes = model.wake
	p.embed(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target == 0 {
		t.Fatal("the vault had no pending vector, and the pass had nothing to stop")
	}
	if embedded == target {
		t.Error("the pass embedded the whole backlog while a note was written")
	}
	select {
	case <-model.wake:
	default:
		t.Error("the wake of the note was taken and not put back")
	}
}

// The pass for a note written waits for the quiet period, and a second write
// puts it off again.
func TestAWakeForNotesIsLeftOnceWhateverTheNumberOfChanges(t *testing.T) {
	worker := NewWorker(Config{})
	for range 3 {
		worker.WakeNotes()
	}
	if got := len(worker.wakes.notes); got != 1 {
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
	case <-worker.wakes.read:
	default:
		t.Fatal("a change of versions woke nothing")
	}
	got := session.drain()
	if len(got) != 2 || got["books/scan.pdf"].ID != "v1" || got["talks/a.mp3"].ID != "v1" {
		t.Fatalf("waiting to be cut: %v, want each changed file once", got)
	}
}

func testVersionChangeEmbedsPendingVaultContent(t *testing.T, openReading openFunc) {
	t.Helper()
	model := &fakeEmbedder{dims: 64}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	p := newPass(made, db, newRecorder(), model, model.GetModel().Recipe(), v)
	p.session.ReportVersionsChanged(v, "books/scan.pdf")
	<-p.worker.wakes.read
	p.embedRecognized(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target == 0 || embedded != target {
		t.Errorf("after a version change %d of %d chunks carry a vector", embedded, target)
	}
}

func testVersionChangeOfANoteKeepsItsChunks(t *testing.T, openReading openFunc) {
	t.Helper()
	model := &fakeEmbedder{dims: 64}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)
	unembedded, _, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if unembedded == 0 {
		t.Fatal("the note was cut into no chunk that needs a vector")
	}

	p := newPass(made, db, newRecorder(), model, model.GetModel().Recipe(), v)
	p.session.ReportVersionsChanged(v, "Note.md")
	<-p.worker.wakes.read
	p.embedRecognized(t.Context())

	target, embedded, err := db.Progress.GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if target != unembedded || embedded != target {
		t.Errorf("after a version change of the note %d of %d chunks carry a vector, want %d of %d",
			embedded, target, unembedded, unembedded)
	}
}

// blocking is a model whose every call runs until its context ends, as a call
// to an embedding plugin does on a long book.
type blocking struct {
	*fakeEmbedder
	started chan struct{}
	stopped chan error
}

func (b *blocking) Embed(ctx context.Context, _ []string) ([][]float32, error) {
	signal(b.started)
	<-ctx.Done()
	select {
	case b.stopped <- ctx.Err():
	default:
	}
	return nil, ctx.Err()
}

func (*blocking) Concurrency() int { return 1 }

func testCancelingTheVectorTaskEndsTheEmbedCall(t *testing.T, openReading openFunc) {
	t.Helper()
	events := newRecorder()
	model := &blocking{
		fakeEmbedder: &fakeEmbedder{dims: 64},
		started:      make(chan struct{}, 1),
		stopped:      make(chan error, 1),
	}
	made, db := openReading(t)
	fit(t, db, model)

	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)

	ended := make(chan struct{})
	go func() {
		defer close(ended)
		newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())
	}()

	select {
	case <-model.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass never called the model")
	}
	if !events.tasks.Cancel(work.MakingVectors) {
		t.Fatalf("the running vector pass is not cancellable: %+v", events.tasks.List())
	}

	select {
	case err := <-model.stopped:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("the embed call ended with %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the embed call in flight was not canceled")
	}
	select {
	case <-ended:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass did not return after its task was canceled")
	}
	if pos := findTask(t, events, work.MakingVectors); pos != nil {
		t.Errorf("the canceled pass is still listed: %+v", *pos)
	}
}

// WorkerIndex is the part of an open index the tests of the worker read and write.
type WorkerIndex struct {
	Vaults      vaultport.VaultRepository
	Notes       noteport.NoteRepository
	FileStats   noteport.FileStats
	Maintenance indexport.IndexMaintenance
	Progress    indexport.IndexProgress
	Vectors     embeddingport.VectorIndex
	Closer      io.Closer
}

// testPass is a pass with the worker it runs for, which a test reaches into.
type testPass struct {
	*pass
	worker *Worker
}
