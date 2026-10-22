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

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

func (r *recorder) ReportTask(at run.Task)                         { r.tasks.Set(at) }
func (r *recorder) FinishTask(id string)                           { r.tasks.Remove(id) }
func (r *recorder) ReportError(error)                              {}
func (r *recorder) ReportNotes(domain.Vault, int)                  {}
func (r *recorder) ReportBooks(domain.Vault, source.ExtractResult) {}

// findTask is one piece of work as the list holds it, or nothing where the list
// does not hold it.
func findTask(t *testing.T, r *recorder, id string) *run.Task {
	t.Helper()
	for _, at := range r.tasks.List() {
		if at.ID == id {
			return &at
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
	t.Cleanup(func() { db.Close() })
	return container.NewEditorAssembly(cfg, nil, db), db
}

// getModelWait blocks until the model behind an embedder has turned up.
func getModelWait(embedder port.Embedder) WaitForModel {
	if waiting, ok := embedder.(embedding.WaitingEmbedder); ok {
		return waiting.Wait
	}
	return nil
}

// newPass is the indexing of one vault over an embedder, reporting into events.
func newPass(
	made container.EditorAssembly, db *container.Index, events Events, embedder port.Embedder, recipe string,
	v domain.Vault,
) *pass {
	worker := NewWorker(Config{
		Reading:  made.ReadWholeVault,
		Embedder: embedder,
		Wait:     getModelWait(embedder),
		Owed: func(ctx context.Context, v domain.Vault) (int64, int64, error) {
			return db.Progress().GetProgress(ctx, v.ID, recipe)
		},
		Events: events,
		Quiet:  time.Hour,
		After:  time.After,
	})
	return &pass{worker: worker, session: worker.NewSession(), v: v}
}

// cut reads the vault's notes, so that its chunks are in the index owing their
// vectors.
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
func fit(t *testing.T, db *container.Index, model port.Embedder) {
	t.Helper()
	if err := db.FitVectors(t.Context(), model.GetModel().Dimensions, model.GetModel().Recipe()); err != nil {
		t.Fatal(err)
	}
}

// asked is a model that answers a direction belonging to the text it was given.
// It reaches nothing outside this process, and counts what it was asked.
type asked struct {
	dims int

	mu    sync.Mutex
	calls int
}

func (a *asked) GetModel() port.EmbeddingModel {
	return port.EmbeddingModel{Name: "fake", Dimensions: a.dims}
}

func (a *asked) Embed(_ context.Context, texts []string) ([][]float32, error) {
	a.mu.Lock()
	a.calls++
	a.mu.Unlock()

	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		out = append(out, direction(text, a.dims))
	}
	return out, nil
}

func (*asked) Close() error { return nil }

func (*asked) Concurrency() int { return 1 }

func (a *asked) times() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

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
	*asked
}

func (*sulking) Embed(context.Context, []string) ([][]float32, error) {
	return nil, errors.New("the model is not answering")
}

func (*sulking) Concurrency() int { return 1 }

// A model that is there and not answering leaves the pass in the list under the
func TestAPassThatCouldNotEmbedStaysInTheList(t *testing.T) {
	model := &sulking{asked: &asked{dims: 64}}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	at := findTask(t, events, MakingVectors)
	if at == nil {
		t.Fatal("the pass that could not embed took itself out of the list")
	}
	if at.Error == "" {
		t.Errorf("the pass is shown as running: %+v", *at)
	}
}

// What ended is not what is being done, and a list a finished pass stays in is a
// list nobody reads.
func TestAPassThatEmbeddedPreservesTheList(t *testing.T) {
	model := &asked{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.GetModel().Recipe(), v).embed(t.Context())

	if at := findTask(t, events, MakingVectors); at != nil {
		t.Errorf("a pass that embedded what was owed is still being done: %+v", *at)
	}
}

// peeking is a model that says what was being done when it was asked for a
// vector, which is what a person watching the corner reads.
type peeking struct {
	*asked
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
	return p.asked.Embed(ctx, texts)
}

func (*peeking) Concurrency() int { return 1 }

// opening is one piece of work as it stood when the first vector was asked for.
func (p *peeking) opening(id string) (run.Task, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, at := range p.while {
		if at.ID == id {
			return at, true
		}
	}
	return run.Task{}, false
}

// Indexing is always of something, and a row carrying the word alone leaves a
// person asking which of their notes it is on.
func TestIndexingNamesTheSourceItIsOn(t *testing.T) {
	events := newRecorder()
	watching := &peeking{asked: &asked{dims: 64}, tasks: events.tasks}
	made, db := reading(t)
	fit(t, db, watching)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	newPass(made, db, events, watching, watching.GetModel().Recipe(), v).embed(t.Context())

	at, target := watching.opening(MakingVectors)
	if !target {
		t.Fatal("nothing was being indexed while a vector was asked for")
	}
	if at.About != "Note.md" {
		t.Errorf("the pass says it is indexing %q", at.About)
	}
}

// Every path into the pass runs on the one goroutine that also reads the books
// and cuts what a recognition wrote, and a model is fetched and compiled in
// minutes. A vault that owes nothing holds that goroutine for nothing.
func TestAVaultOwingNoVectorWaitsForNoModel(t *testing.T) {
	model := &asked{dims: 64}
	made, db := reading(t)

	// The notes are on disk and nothing has cut them, so the index holds no
	// chunk and owes no vector.
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})

	// The weights are still coming down, and in this test they never land.
	incoming := embedding.NewDeferredEmbedder(model.GetModel())

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		newPass(made, db, newRecorder(), incoming.GetWaitingEmbedder(), model.GetModel().Recipe(), v).
			embed(t.Context())
	}()

	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass is waiting for a model to embed nothing with")
	}
}

// Fetching a model is not indexing. A row that calls it by the name of the work
// that follows sits at no share of nothing while the weights come down.
func TestNothingIsIndexedWhileTheModelIsOnItsWay(t *testing.T) {
	model := &asked{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)
	events := newRecorder()

	incoming := embedding.NewDeferredEmbedder(model.GetModel())
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		newPass(made, db, events, incoming.GetWaitingEmbedder(), model.GetModel().Recipe(), v).
			embed(t.Context())
	}()

	for range 20 {
		if at := findTask(t, events, MakingVectors); at != nil {
			t.Fatalf("a model still on its way is shown as indexing: %+v", *at)
		}
		time.Sleep(time.Millisecond)
	}

	incoming.SetModel(model, nil)
	<-doneCh

	if at := findTask(t, events, MakingVectors); at != nil {
		t.Errorf("the pass that embedded what was owed is still being done: %+v", *at)
	}
	if model.times() == 0 {
		t.Error("the model landed and nothing was embedded")
	}
}

// A library that would not be read is a vault searching less than it holds, and
// the pass is left standing under what stopped it.
func TestBooksThatCouldNotBeReadStayInTheList(t *testing.T) {
	made, db := reading(t)
	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	events := newRecorder()
	// The index the pass asks what owes its text is gone, which is what it looks
	// like from here when the answer cannot be had.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	newPass(made, db, events, nil, "", v).readBooks(t.Context(), false)

	at := findTask(t, events, ExtractingBooks)
	if at == nil {
		t.Fatal("the pass that could not read the books took itself out of the list")
	}
	if at.Error == "" {
		t.Errorf("the pass is shown as running: %+v", *at)
	}
}

// overheard is a model that keeps the list of what is being done as it stood
// every time it was asked anything, which is what a person watching the corner
// reads.
type overheard struct {
	*asked
	tasks *run.Tasks

	mu   sync.Mutex
	seen [][]run.Task
}

// GetModel is asked at the head of every pass, before a source has been opened.
func (o *overheard) GetModel() port.EmbeddingModel {
	o.keep()
	return o.asked.GetModel()
}

func (o *overheard) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	o.keep()
	return o.asked.Embed(ctx, texts)
}

func (*overheard) Concurrency() int { return 1 }

func (o *overheard) keep() {
	o.mu.Lock()
	o.seen = append(o.seen, o.tasks.List())
	o.mu.Unlock()
}

func (o *overheard) lists() [][]run.Task {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.seen
}

// The row a person watches for minutes must say what it is on. A pass that
// enters the list before it has opened a source stands there as the word
// "Indexing" over an empty line, at a share of nothing, for as long as the first
// vectors take to come back.
func TestIndexingIsNeverAWordWithNothingBelowIt(t *testing.T) {
	events := newRecorder()
	model := &asked{dims: 64}
	made, db := reading(t)
	fit(t, db, model)

	v := testsupport.NewVault(t, map[string]string{"Note.md": noteWith(before, 200)})
	cut(t, made, db, v)

	// A provider that answers over a network is here the moment it is made, so
	// there is no arrival to wait for and the pass begins at once.
	overVal := &overheard{asked: model, tasks: events.tasks}
	target := embedding.NewDeferredEmbedder(model.GetModel())
	target.SetModel(overVal, nil)

	newPass(made, db, events, target.GetWaitingEmbedder(), model.GetModel().Recipe(), v).embed(t.Context())

	for _, list := range overVal.lists() {
		for _, at := range list {
			if at.ID != MakingVectors || at.Error != "" {
				continue
			}
			if at.About == "" {
				t.Errorf("the pass stood in the list saying nothing but %q", at.Title)
			}
			if at.Total > 0 && at.Count == 0 {
				t.Errorf("the pass drew a share of nothing: %+v", at)
			}
		}
	}
}

// nudging is a model that has a recognition write a batch of pages while the
// vector pass is running, and holds the pass until the wake has stopped it.
type nudging struct {
	*asked
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
	return n.asked.Embed(ctx, texts)
}

func (*nudging) Concurrency() int { return 1 }

// A recognition writes a batch of pages every few minutes, and the pages it
// wrote are cut on the goroutine the vector pass runs on. A vault owing a
// hundred thousand vectors owes them for hours.
func TestARecognitionStopsTheVectorPass(t *testing.T) {
	events := newRecorder()
	model := &nudging{asked: &asked{dims: 64}, wake: make(chan struct{}, 1)}
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
		t.Fatal("the vault owed no vector, and the pass had nothing to stand aside from")
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

	if at := findTask(t, events, MakingVectors); at != nil && at.Error != "" {
		t.Errorf("standing aside is shown as a failure: %q", at.Error)
	}

	// Taken up again, the pass asks the index what still owes a vector.
	again := newPass(made, db, events, model.asked, model.GetModel().Recipe(), v)
	again.embed(t.Context())

	target, embedded, err = db.Progress().GetProgress(t.Context(), v.ID, model.GetModel().Recipe())
	if err != nil {
		t.Fatal(err)
	}
	if embedded != target {
		t.Errorf("the pass taken up again left %d of %d chunks without a vector", target-embedded, target)
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
