package recognition

import (
	"context"
	"errors"
	"image"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// reads nothing, and says it was closed.
type blank struct{ isClosed bool }

func (blank) GetRecognitionModel() port.RecognitionModel {
	return port.RecognitionModel{Recogniser: "blank"}
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
	tasks *run.Tasks
	held  *blank

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
	w := &watched{tasks: tasks, held: &blank{}}
	w.Worker = NewWorker(ctx, Config{
		Readers:   vaults{},
		Derived:   shelves{newShelf()},
		Sources:   newStore(),
		Documents: documents{},
		Tasks:     tasks,
		Wait:      waitForContext,
		Proofreading: run.ProofreadingConfig{
			Queue: func(string) (port.ProofreadQueue, error) { return nil, nil },
		},
		Runtime: Runtime{
			Ready: func() bool { return true },
			Open: func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
				w.mu.Lock()
				w.open++
				w.while = tasks.List()
				w.mu.Unlock()
				if why != nil {
					return nil, nil, why
				}
				return w.held, w.held.Close, nil
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
	held := w.tasks.List()
	if len(held) == 0 {
		return run.Task{}, false
	}
	if len(held) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
	return held[0], true
}

var somewhere = domain.Vault{ID: "v", Path: "/nowhere"}

// One at a time: the models hold a worker each. A document named while one is
// being read waits its turn, and a document named twice waits once.
func TestADocumentNamedWhileOneIsBeingReadWaitsItsTurn(t *testing.T) {
	w := newWatched(t, errors.New("nothing to read with"))

	reading, held := make(chan struct{}, 2), make(chan struct{})
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		w.mu.Lock()
		w.open++
		w.mu.Unlock()
		reading <- struct{}{}
		<-held
		return nil, nil, errors.New("nothing to read with")
	}

	if got := w.Start(somewhere, "a.pdf"); got != port.Began {
		t.Fatalf("the first document was not read: %v", got)
	}
	// The first document is out of the line and being read.
	<-reading

	if got := w.Start(somewhere, "b.pdf"); got != port.Queued {
		t.Errorf("a second document was read while one was being read: %v", got)
	}
	if got := w.Start(somewhere, "b.pdf"); got != port.Queued {
		t.Errorf("the same document named again: %v", got)
	}
	if w.CountWaiting() != 1 {
		t.Errorf("%d documents are in line", w.CountWaiting())
	}

	close(held)
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

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	at, held := w.getTask(t)
	if !held {
		t.Fatal("the failure was not said")
	}
	if at.ID != "recognition-a.pdf" || at.Error == "" || at.About != "a.pdf" {
		t.Errorf("got %+v", at)
	}
}

// Getting the models is a step of its own, and it is named for what it is. A
// row that calls it by the name of the work that follows leaves a person
// watching a recognition that has not begun.
func TestTheModelsAreGotUnderTheirOwnName(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	at := w.opening(t)
	if at.Title != "Fetching models" {
		t.Errorf("getting the models is shown as %q", at.Title)
	}
	// Nothing has come down, so there is no share of it to draw.
	if at.Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", at.Total)
	}
}

// Retrying a failed recognition replaces the failure with progress and removes the
// task upon success.
func TestRetryingAFailedRecognitionClearsTheTaskOnSuccess(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	at, held := w.getTask(t)
	if !held || at.ID != "recognition-a.pdf" || at.Error == "" {
		t.Fatalf("first failure not recorded properly: %+v", at)
	}

	// Now runtime and reader succeed.
	lib := newLibrary()
	lib.hold("a.pdf", domain.KindBook, []byte("pdf"), 1)
	w.Worker.with.Readers = vaults{somewhere.ID: lib}
	w.Worker.with.Documents = documents{}
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		return w.held, w.held.Close, nil
	}

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the retry was not started")
	}
	w.waitUntilDone(t)

	if at, held := w.getTask(t); held {
		t.Errorf("task remained in list after successful retry: %+v", at)
	}
}

// Each document's failure is tracked under its own name.
func TestMultipleFailedRecognitionsAreTrackedIndependently(t *testing.T) {
	w := newWatched(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the first document was not read")
	}
	w.waitUntilDone(t)

	if w.Start(somewhere, "b.pdf") != port.Began {
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
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		cancel()
		return nil, nil, context.Canceled
	}

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	if at, held := w.getTask(t); held {
		t.Errorf("a recognition that was stopped is in the list: %+v", at)
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
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		select {
		case <-abrupt:
			runtime.Goexit()
		default:
		}
		return nil, nil, errors.New("nothing to read with")
	}

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.waitUntilDone(t)

	if w.Start(somewhere, "b.pdf") != port.Began {
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
	if !w.held.isClosed {
		t.Error("what was opened was not given back")
	}
}

// A recognition writes to the index, and the application waits for it before what
// it writes to is closed.
func TestTheApplicationWaitsForARecognitionItStarted(t *testing.T) {
	w := newWatched(t, nil)

	holding := make(chan struct{})
	w.Worker.with.Runtime.Open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		<-holding
		return nil, nil, errors.New("nothing to read with")
	}
	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}

	waited := make(chan struct{})
	go func() { w.Wait(); close(waited) }()

	select {
	case <-waited:
		t.Fatal("the wait ended while the recognition was still running")
	case <-time.After(50 * time.Millisecond):
	}

	close(holding)
	select {
	case <-waited:
	case <-time.After(2 * time.Second):
		t.Fatal("the wait never ended")
	}
}

// A queue that failed to build is said, as the proofreader that failed to build
// is said. A person who configured a queue and is given none is owed the reason.
func TestAProofreadQueueThatFailedToBuildIsSaid(t *testing.T) {
	w := newWatched(t, nil)
	w.Worker.with.Proofreading = run.ProofreadingConfig{
		IsNamed: true, IsAutomatic: true,
		By: func(string) (port.Proofreader, error) { return &puts{}, nil },
		Queue: func(string) (port.ProofreadQueue, error) {
			return nil, errors.New("no queue for the proofreading service")
		},
	}

	w.proofread(t.Context(), somewhere, "a.pdf")

	at, held := w.getTask(t)
	if !held {
		t.Fatal("a queue that failed to build was not said")
	}
	if at.Error == "" || at.About != "a.pdf" {
		t.Errorf("got %+v", at)
	}
}
