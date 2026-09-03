package container

import (
	"context"
	"errors"
	"image"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// reads nothing, and says it was closed.
type blank struct{ closed bool }

func (blank) Recognition() port.Recognition { return port.Recognition{Recogniser: "blank"} }

func (blank) Recognise(context.Context, image.Image) ([]ocr.Block, error) { return nil, nil }

func (b *blank) Close() error {
	b.closed = true
	return nil
}

// watched is a Recognising given what it reads with, and a way to know what it
// said.
type watched struct {
	*Recognising
	tasks *task.Tasks
	held  *blank

	mu   sync.Mutex
	open int
	// while is the list as it stood when the models were asked for.
	while []task.Task
}

func recognising(t *testing.T, why error) *watched {
	t.Helper()
	tasks := task.New()
	w := &watched{tasks: tasks, held: &blank{}}
	w.Recognising = &Recognising{
		cfg:   Config{ServiceDir: ".numen"},
		tasks: tasks,
		ready: func() bool { return true },
		queue: func() (port.ProofreadQueue, error) { return nil, nil },
		open: func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
			w.mu.Lock()
			w.open++
			w.while = tasks.List()
			w.mu.Unlock()
			if why != nil {
				return nil, nil, why
			}
			return w.held, w.held.Close, nil
		},
	}
	return w
}

// opened is how many times a recogniser was asked for.
func (w *watched) opened() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.open
}

// opening is the one piece of work in the list while the models were asked for.
func (w *watched) opening(t *testing.T) task.Task {
	t.Helper()
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.while) != 1 {
		t.Fatalf("the list holds %d pieces of work while the models arrive: %+v", len(w.while), w.while)
	}
	return w.while[0]
}

// settled waits for the reading to be over.
func (w *watched) settled(t *testing.T) {
	t.Helper()
	for range 200 {
		if !w.Running() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("the reading never ended")
}

// said is the one task in the list, or nothing.
func (w *watched) said(t *testing.T) (task.Task, bool) {
	t.Helper()
	held := w.tasks.List()
	if len(held) == 0 {
		return task.Task{}, false
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
	w := recognising(t, errors.New("nothing to read with"))

	reading, held := make(chan struct{}, 2), make(chan struct{})
	w.Recognising.open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
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
	if w.Waiting() != 1 {
		t.Errorf("%d documents are in line", w.Waiting())
	}

	close(held)
	w.settled(t)

	if w.opened() != 2 {
		t.Errorf("a recogniser was opened %d times", w.opened())
	}
	if w.Waiting() != 0 {
		t.Errorf("%d documents were left in line", w.Waiting())
	}
}

// A failure nobody was shown is a failure nobody can act on.
func TestAReadingThatFailedStaysInTheList(t *testing.T) {
	w := recognising(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.settled(t)

	at, held := w.said(t)
	if !held {
		t.Fatal("the failure was not said")
	}
	if at.Failed == "" || at.About != "a.pdf" {
		t.Errorf("got %+v", at)
	}
}

// Getting the models is a step of its own, and it is named for what it is. A
// row that calls it by the name of the work that follows leaves a person
// watching a reading that has not begun.
func TestTheModelsAreGotUnderTheirOwnName(t *testing.T) {
	w := recognising(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.settled(t)

	at := w.opening(t)
	if at.Doing != "Fetching models" {
		t.Errorf("getting the models is shown as %q", at.Doing)
	}
	// Nothing has come down, so there is no share of it to draw.
	if at.Total != 0 {
		t.Errorf("a step that has counted nothing is drawn against %d", at.Total)
	}
}

// The next reading takes the one before it out of the list: one reading is one
// line, however many have failed.
func TestTheNextReadingClearsTheOneBeforeIt(t *testing.T) {
	w := recognising(t, errors.New("no models on this machine"))

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.settled(t)
	if _, held := w.said(t); !held {
		t.Fatal("the first failure was not said")
	}

	if w.Start(somewhere, "b.pdf") != port.Began {
		t.Fatal("the second document was not read")
	}
	w.settled(t)

	at, held := w.said(t)
	if !held {
		t.Fatal("the second failure was not said")
	}
	if at.About != "b.pdf" {
		t.Errorf("the list holds %+v", at)
	}
	if w.opened() != 2 {
		t.Errorf("a recogniser was opened %d times", w.opened())
	}
}

// A reading whoever asked for it stopped is a reading that is over, and not one
// that failed.
func TestAReadingStoppedIsNotAFailure(t *testing.T) {
	w := recognising(t, nil)
	ctx, cancel := context.WithCancel(t.Context())
	w.Recognising.under = ctx
	w.Recognising.open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
		cancel()
		return nil, nil, context.Canceled
	}

	if w.Start(somewhere, "a.pdf") != port.Began {
		t.Fatal("the document was not read")
	}
	w.settled(t)

	if at, held := w.said(t); held {
		t.Errorf("a reading that was stopped is in the list: %+v", at)
	}
}

// A reading that ends where nothing expected it to holds nothing afterwards:
// the next document is taken.
func TestAReadingThatEndsAbruptlyDoesNotHoldTheNextOne(t *testing.T) {
	w := recognising(t, nil)

	// The first reading leaves its goroutine partway down, as a panic under the
	// reading does.
	abrupt := make(chan struct{}, 1)
	abrupt <- struct{}{}
	w.Recognising.open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
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
	w.settled(t)

	if w.Start(somewhere, "b.pdf") != port.Began {
		t.Fatal("nothing was read after a reading that ended where nothing expected it to")
	}
	w.settled(t)
}

// Every page is read through the runtime the process made before its window. A
// reading through one made afterwards writes down a blank page for every page of
// the document, so what was missing is fetched and nothing is read.
func TestNothingIsReadThroughARuntimeMadeAfterTheWindow(t *testing.T) {
	w := recognising(t, nil)
	w.Recognising.standing = func() bool { return false }

	err := w.read(t.Context(), somewhere, "reading-1", "a.pdf")
	if !errors.Is(err, errLateRuntime) {
		t.Fatalf("the document was read through a runtime made after the window: %v", err)
	}
	if w.opened() != 1 {
		t.Errorf("what was missing was fetched %d times", w.opened())
	}
	if !w.held.closed {
		t.Error("what was opened was not given back")
	}
}

// A reading writes to the index, and the application waits for it before what
// it writes to is closed.
func TestTheApplicationWaitsForAReadingItStarted(t *testing.T) {
	w := recognising(t, nil)

	holding := make(chan struct{})
	w.Recognising.open = func(context.Context, func(string, int64, int64)) (port.Recogniser, func() error, error) {
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
		t.Fatal("the wait ended while the reading was still running")
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
	t.Setenv(proofreading.KeyEnvVar, "sk-test")
	w := recognising(t, nil)

	service := proofreading.ServiceDefaults()
	service.Name = "a-model"

	cfg := Config{ServiceDir: ".numen"}
	cfg.Proofreading = proofreading.Defaults()
	cfg.Proofreading.Profiles = map[string]proofreading.Profile{"a-service": service}
	cfg.ScanProofreading = proofreading.Proofread{With: "a-service", Automatically: true}
	w.Recognising.cfg = cfg
	w.Recognising.queue = func() (port.ProofreadQueue, error) {
		return nil, errors.New("no queue for the proofreading service")
	}

	w.correct(t.Context(), somewhere, "a.pdf")

	at, held := w.said(t)
	if !held {
		t.Fatal("a queue that failed to build was not said")
	}
	if at.Failed == "" || at.About != "a.pdf" {
		t.Errorf("got %+v", at)
	}
}
