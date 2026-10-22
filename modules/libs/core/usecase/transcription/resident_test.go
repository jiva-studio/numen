package transcription

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/transcriptionport"
)

// counted is a runtime that counts how many times the models were opened and
// closed, and can fail on request.
type counted struct {
	provider *deaf

	mu       sync.Mutex
	opened   int
	closed   int
	openFail error
}

func (c *counted) open(context.Context, func(string, int64, int64)) (transcriptionport.Transcriber, func() error, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.openFail != nil {
		return nil, nil, c.openFail
	}
	c.opened++
	return c.provider, func() error {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.closed++
		return nil
	}, nil
}

func (c *counted) counts() (opened, closed int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.opened, c.closed
}

func newResidentWorker(t *testing.T, keep bool, recordings ...string) (*Worker, *counted, unheard, domain.Vault) {
	t.Helper()
	delta := &deaf{}
	listening, v := newTranscriptionWorker(t, delta, recordings...)
	models := &counted{provider: delta}
	listening.with.Runtime.Open = models.open
	listening.with.Runtime.CanKeep = func() bool { return keep }
	return listening, models, unheard{recordings: recordings}, v
}

func TestARoundOpensTheModelsOnceAndClosesThemAtTheEnd(t *testing.T) {
	listening, models, recognized, v := newResidentWorker(t, true, "a.mp3", "b.mp3", "c.mp3")

	listening.round(t.Context(), recognized, v)

	if opened, closed := models.counts(); opened != 1 || closed != 1 {
		t.Errorf("opened %d, closed %d", opened, closed)
	}
	if got := models.provider.times(); got != 3 {
		t.Errorf("%d recordings were heard", got)
	}
	if isHeld(listening.with.Models) {
		t.Error("the models lock is still held")
	}
}

func TestModelsAreOpenedForEachRecordingWhenTheyCannotStayLoaded(t *testing.T) {
	listening, models, recognized, v := newResidentWorker(t, false, "a.mp3", "b.mp3", "c.mp3")

	listening.round(t.Context(), recognized, v)

	if opened, closed := models.counts(); opened != 3 || closed != 3 {
		t.Errorf("opened %d, closed %d", opened, closed)
	}
}

func TestModelsAreOpenedForEachRecordingWhenNoPredicateIsGiven(t *testing.T) {
	listening, models, recognized, v := newResidentWorker(t, true, "a.mp3", "b.mp3")
	listening.with.Runtime.CanKeep = nil

	listening.round(t.Context(), recognized, v)

	if opened, _ := models.counts(); opened != 2 {
		t.Errorf("opened %d", opened)
	}
}

func TestModelsAreReleasedForARunWaitingOnTheLock(t *testing.T) {
	listening, models, recognized, v := newResidentWorker(t, true, "a.mp3", "b.mp3")

	granted := make(chan func(), 1)
	listening.with.Runtime.Open = func(ctx context.Context, tell func(string, int64, int64)) (transcriptionport.Transcriber, func() error, error) {
		delta, letGo, err := models.open(ctx, tell)
		if opened, _ := models.counts(); opened == 1 {
			// A scan joins the line while the first recording is being heard.
			go func() {
				release, err := listening.with.Models.Acquire(t.Context(), false, nil)
				if err == nil {
					granted <- release
				}
			}()
			for !listening.with.Models.HasWaiting() {
				time.Sleep(time.Millisecond)
			}
		}
		return delta, letGo, err
	}

	isDone := make(chan struct{})
	go func() { listening.round(t.Context(), recognized, v); close(isDone) }()

	release := <-granted
	if _, closed := models.counts(); closed != 1 {
		t.Errorf("the models were not closed before the waiter took the lock: %d", closed)
	}
	release()
	<-isDone

	if opened, closed := models.counts(); opened != 2 || closed != 2 {
		t.Errorf("opened %d, closed %d", opened, closed)
	}
}

func TestAnOpenFailureReleasesTheLockAndPropagates(t *testing.T) {
	listening, models, _, v := newResidentWorker(t, true, "a.mp3")
	models.openFail = errors.New("no runtime")

	err := listening.transcribeOne(t.Context(), v, "a.mp3", false)

	if !errors.Is(err, errNothingTranscribes) {
		t.Errorf("the failure was %v", err)
	}
	if isHeld(listening.with.Models) {
		t.Error("the lock is still held")
	}
}

func TestAFailureMidRoundReleasesTheModels(t *testing.T) {
	listening, models, _, v := newResidentWorker(t, true, "a.mp3")
	listening.with.Readers = vaults{}

	_ = listening.transcribeOne(t.Context(), v, "a.mp3", false)

	if opened, closed := models.counts(); opened != 1 || closed != 1 {
		t.Errorf("opened %d, closed %d", opened, closed)
	}
	if isHeld(listening.with.Models) {
		t.Error("the lock is still held")
	}
}

func TestCancellationReleasesTheModels(t *testing.T) {
	listening, models, recognized, v := newResidentWorker(t, true, "a.mp3", "b.mp3")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	open := listening.with.Runtime.Open
	listening.with.Runtime.Open = func(c context.Context, tell func(string, int64, int64)) (transcriptionport.Transcriber, func() error, error) {
		cancel()
		return open(c, tell)
	}

	listening.round(ctx, recognized, v)

	if opened, closed := models.counts(); opened != closed {
		t.Errorf("opened %d, closed %d", opened, closed)
	}
	if isHeld(listening.with.Models) {
		t.Error("the lock is still held")
	}
}

func TestIdleModelsAreReleasedAfterTheIdlePeriod(t *testing.T) {
	listening, models, _, v := newResidentWorker(t, true, "a.mp3")
	listening.residentIdle = 10 * time.Millisecond
	listening.with.Delay = func(_ time.Duration, f func()) func() bool {
		go f()
		return func() bool { return false }
	}

	if err := listening.transcribeOne(t.Context(), v, "a.mp3", false); err != nil {
		t.Fatal(err)
	}
	if _, closed := models.counts(); closed != 0 {
		t.Fatalf("closed %d before the idle period", closed)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, closed := models.counts(); closed == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the idle models were not closed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if isHeld(listening.with.Models) {
		t.Error("the lock is still held")
	}
}

// isHeld says whether a run holds the lock: a free lock is handed over at once,
// and a held one is waited for until the deadline.
func isHeld(g *run.Lock) bool {
	ctx, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	release, err := g.Acquire(ctx, true, nil)
	if err != nil {
		return true
	}
	release()
	return false
}
