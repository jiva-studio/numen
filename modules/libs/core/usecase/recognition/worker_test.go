package recognition

import (
	"context"
	"errors"
	"image"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/modules/libs/core/port/windowport"
)

// reads nothing, and says it was closed.
type blank struct{ isClosed bool }

func (blank) GetRecognitionModel() recognitionport.RecognitionModel {
	return recognitionport.RecognitionModel{Recogniser: "blank"}
}

func (blank) Recognise(context.Context, image.Image) ([]domain.OCRBlock, error) { return nil, nil }

func (b *blank) Close() error {
	b.isClosed = true
	return nil
}

// watched is a Worker given what it reads with, and a way to know what it
// said.
type watched struct {
	*Worker
	tasks      *run.Tasks
	activeItem *blank

	mu   sync.Mutex
	open int
	// while is the list as it stood when the models were asked for.
	while []run.Task
}

func newWatched(t *testing.T, why error) *watched {
	t.Helper()
	return newWatchedUnder(t, t.Context(), why)
}

// newWatchedUnder is a watched worker whose runs end with ctx.
func newWatchedUnder(t *testing.T, ctx context.Context, why error) *watched {
	t.Helper()
	tasks := run.NewTasks()
	w := &watched{tasks: tasks, activeItem: &blank{}}
	w.Worker = NewWorker(ctx, Config{
		Readers:   vaults{},
		Derived:   fakeStores{newShelf()},
		Sources:   newStore(),
		Documents: documents{},
		Tasks:     tasks,
		Runtime: Runtime{
			Ready: func() bool { return true },
			Open: func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
				w.mu.Lock()
				w.open++
				w.while = tasks.List()
				w.mu.Unlock()
				if why != nil {
					return nil, nil, why
				}
				return w.activeItem, w.activeItem.Close, nil
			},
		},
	})
	return w
}

// countOpens is how many times a recogniser was asked for.
func (w *watched) countOpens() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.open
}

// opening is the one piece of work in the list while the models were asked for.
func (w *watched) opening(t *testing.T) run.Task {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.while) != 1 {
		t.Fatalf("the list holds %d pieces of work while the models arrive: %+v", len(w.while), w.while)
	}
	return w.while[0]
}

// waitUntilDone waits for the recognition to be over.
func (w *watched) waitUntilDone(t *testing.T) {
	t.Helper()
	for range 200 {
		if !w.IsRunning() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the recognition never ended")
}

// getTask is the one task in the list, or nothing.
func (w *watched) getTask(t *testing.T) (run.Task, bool) {
	t.Helper()
	target := w.tasks.List()
	if len(target) == 0 {
		return run.Task{}, false
	}
	if len(target) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(target), target)
	}
	return target[0], true
}

var somewhere = domain.Vault{ID: "v", Path: "/nowhere"}

// One at a time: the models hold a worker each. A document named while one is
// being read waits its turn, and a document named twice waits once.
func TestADocumentNamedWhileOneIsBeingReadWaitsItsTurn(t *testing.T) {
	w := newWatched(t, errors.New("nothing to read with"))

	reading, target := make(chan struct{}, 2), make(chan struct{})
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
		w.mu.Lock()
		w.open++
		w.mu.Unlock()
		reading <- struct{}{}
		<-target
		return nil, nil, errors.New("nothing to read with")
	}

	if got := w.Start(somewhere, "a.pdf"); got != windowport.Began {
		t.Fatalf("the first document was not read: %v", got)
	}
	// The first document is out of the line and being read.
	<-reading

	if got := w.Start(somewhere, "b.pdf"); got != windowport.Queued {
		t.Errorf("a second document was read while one was being read: %v", got)
	}
	if got := w.Start(somewhere, "b.pdf"); got != windowport.Queued {
		t.Errorf("the same document named again: %v", got)
	}
	if w.CountWaiting() != 1 {
		t.Errorf("%d documents are in line", w.CountWaiting())
	}

	close(target)
	w.waitUntilDone(t)

	if w.countOpens() != 2 {
		t.Errorf("a recogniser was opened %d times", w.countOpens())
	}
	if w.CountWaiting() != 0 {
		t.Errorf("%d documents were left in line", w.CountWaiting())
	}
}

// A failure nobody was shown is a failure nobody can act on.
func TestARecognitionThatFailedStaysInTheList(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	pos, target := w.getTask(t)
	if !target {
		t.Fatal("the failure was not said")
	}
	if pos.ID != "recognition-a.pdf" || pos.Error == "" || pos.About != "a.pdf" {
		t.Errorf("got %+v", pos)
	}
}

// Getting the models is a step of its own, and it is named for what it is. A
// row that calls it by the name of the work that follows leaves a person
// watching a recognition that has not begun.
func TestTheModelsAreGotBelowTheirOwnName(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	pos := w.opening(t)
	if pos.Title != "Fetching models" {
		t.Errorf("getting the models is shown as %q", pos.Title)
	}
	// Nothing has come down, so there is no share of it to draw.
	if pos.Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", pos.Total)
	}
}

// Retrying a failed recognition replaces the failure with progress and removes the
// task upon success.
func TestRetryingAFailedRecognitionClearsTheTaskOnSuccess(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	pos, target := w.getTask(t)
	if !target || pos.ID != "recognition-a.pdf" || pos.Error == "" {
		t.Fatalf("first failure not recorded properly: %+v", pos)
	}

	// Now runtime and reader succeed.
	lib := newLibrary()
	lib.hold("a.pdf", domain.KindBook, []byte("pdf"), 1)
	w.Worker.with.Readers = vaults{somewhere.ID: lib}
	w.Worker.with.Documents = documents{}
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
		return w.activeItem, w.activeItem.Close, nil
	}

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the retry was not started")
	}
	w.waitUntilDone(t)

	if offset, targetVal := w.getTask(t); targetVal {
		t.Errorf("task remained in list after successful retry: %+v", offset)
	}
}

// Each document's failure is tracked under its own name.
func TestMultipleFailedRecognitionsAreTrackedIndependently(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the first document was not read")
	}
	w.waitUntilDone(t)

	if w.Start(somewhere, "b.pdf") != windowport.Began {
		t.Fatal("the second document was not read")
	}
	w.waitUntilDone(t)

	list := w.tasks.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %+v", len(list), list)
	}
	if list[0].ID != "recognition-a.pdf" || list[1].ID != "recognition-b.pdf" {
		t.Errorf("unexpected task list: %+v", list)
	}
}

// A recognition whoever asked for it stopped is a recognition that is over, and not one
// that failed.
func TestARecognitionStoppedIsNotAFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w := newWatchedUnder(t, ctx, nil)
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
		cancel()
		return nil, nil, context.Canceled
	}

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	if pos, target := w.getTask(t); target {
		t.Errorf("a recognition that was stopped is in the list: %+v", pos)
	}
}

// A recognition that ends where nothing expected it to holds nothing afterwards:
// the next document is taken.
func TestARecognitionThatEndsAbruptlyDoesNotHoldTheNextOne(t *testing.T) {
	w := newWatched(t, nil)

	// The first recognition leaves its goroutine partway down, as a panic under the
	// reading does.
	abrupt := make(chan struct{}, 1)
	abrupt <- struct{}{}
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
		select {
		case <-abrupt:
			runtime.Goexit()
		default:
		}
		return nil, nil, errors.New("nothing to read with")
	}

	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	if w.Start(somewhere, "b.pdf") != windowport.Began {
		t.Fatal("nothing was read after a recognition that ended where nothing expected it to")
	}
	w.waitUntilDone(t)
}

// Every page is read through the runtime the process made before its window. A
// recognition through one made afterwards writes down a blank page for every page of
// the document, so what was missing is fetched and nothing is read.
func TestNothingIsReadThroughARuntimeMadeAfterTheWindow(t *testing.T) {
	w := newWatched(t, nil)
	w.Worker.with.Runtime.Prepared = func() bool { return false }

	err := w.recognise(t.Context(), somewhere, "recognition-1", "a.pdf")
	if !errors.Is(err, errLateRuntime) {
		t.Fatalf("the document was read through a runtime made after the window: %v", err)
	}
	if w.countOpens() != 1 {
		t.Errorf("what was missing was fetched %d times", w.countOpens())
	}
	if !w.activeItem.isClosed {
		t.Error("what was opened was not given back")
	}
}

// A recognition writes to the index, and the application waits for it before what
// it writes to is closed.
func TestTheApplicationWaitsForARecognitionItStarted(t *testing.T) {
	w := newWatched(t, nil)

	blockCh := make(chan struct{})
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (recognitionport.Recogniser, func() error, error) {
		<-blockCh
		return nil, nil, errors.New("nothing to read with")
	}
	if w.Start(somewhere, "a.pdf") != windowport.Began {
		t.Fatal("the document was not read")
	}

	waited := make(chan struct{})
	go func() { w.Wait(); close(waited) }()

	select {
	case <-waited:
		t.Fatal("the wait ended while the recognition was still running")
	case <-time.After(50 * time.Millisecond):
	}

	close(blockCh)
	select {
	case <-waited:
	case <-time.After(2 * time.Second):
		t.Fatal("the wait never ended")
	}
}

// A proofreader that failed to open is said. A person who configured one and is
// given none is owed the reason.
func TestAProofreaderThatFailedToOpenIsSaid(t *testing.T) {
	w := newWatched(t, nil)
	w.Worker.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func() (recognitionport.Proofreader, error) {
			return nil, errors.New("proofreading plugin is unavailable")
		},
	}

	w.proofread(t.Context(), somewhere, "a.pdf")

	pos, target := w.getTask(t)
	if !target {
		t.Fatal("a proofreader that failed to open was not said")
	}
	if pos.Error == "" || pos.About != "a.pdf" {
		t.Errorf("got %+v", pos)
	}
}

// A proofreading the person has not let ask a model says what allows it, in the
// list of what is being done.
func TestAProofreadingWithoutConsentReturnsWhereToAllowIt(t *testing.T) {
	w := newWatched(t, nil)
	w.Worker.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func() (recognitionport.Proofreader, error) {
			return nil, errors.Join(errors.New("proofread rpc"), completionport.ErrNoConsent)
		},
	}

	w.proofread(t.Context(), somewhere, "a.pdf")

	pos, isFound := w.getTask(t)
	if !isFound || pos.Error != "AI access not allowed — allow it in Settings → AI" {
		t.Errorf("got %+v", pos)
	}
}
