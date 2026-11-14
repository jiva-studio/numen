package indexing

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/work"
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/sourcefake"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// emitted counts the text steps, vector plans and quiet timers a session had started when the
// index-changed listener received the event of the vault.
type emitted struct {
	vault  domain.VaultID
	texts  int
	plans  int
	timers int
}

// steps is a fake of the index under a session. Each step takes the next result queued for it and
// answers nothing more once the queue is empty. The pass calls it from its own goroutine.
type steps struct {
	mu        sync.Mutex
	read      vaultreading.ScanResult
	texts     []vaultreading.IndexResult
	cuts      []vaultreading.IndexResult
	vectors   []func(ctx context.Context, onProgress func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error)
	isPending bool
	textCalls int
	planCalls int
	timers    []chan time.Time
}

func (s *steps) ScanWithProgress(context.Context, domain.Vault, func(vaultreading.ScanResult)) (vaultreading.ScanResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read, nil
}

func (s *steps) IndexWithProgress(
	_ context.Context, _ domain.Vault, _ func(vaultreading.IndexResult), _ func(path string) bool,
) (vaultreading.IndexResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.textCalls++
	if len(s.texts) == 0 {
		return vaultreading.IndexResult{}, nil
	}
	res := s.texts[0]
	s.texts = s.texts[1:]
	return res, nil
}

func (s *steps) IndexOne(context.Context, domain.Vault, string) (vaultreading.IndexResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cuts) == 0 {
		return vaultreading.IndexResult{}, nil
	}
	res := s.cuts[0]
	s.cuts = s.cuts[1:]
	return res, nil
}

func (s *steps) EmbedWithProgress(
	ctx context.Context, _ domain.Vault, onProgress func(vaultreading.EmbedResult), _ func(path, hash string) bool,
) (vaultreading.EmbedResult, error) {
	s.mu.Lock()
	if len(s.vectors) == 0 {
		s.mu.Unlock()
		return vaultreading.EmbedResult{}, nil
	}
	step := s.vectors[0]
	s.vectors = s.vectors[1:]
	s.mu.Unlock()
	return step(ctx, onProgress)
}

func (s *steps) getVectorCounts(context.Context, domain.Vault) (held, embedded int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.planCalls++
	if s.isPending {
		return 4, 0, nil
	}
	return 4, 4, nil
}

func (s *steps) queueTexts(res vaultreading.IndexResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.texts = append(s.texts, res)
}

func (s *steps) setPending(isPending bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isPending = isPending
}

func (s *steps) startTimer(time.Duration) <-chan time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	timer := make(chan time.Time, 1)
	s.timers = append(s.timers, timer)
	return timer
}

func (s *steps) snapshot(v domain.Vault) emitted {
	s.mu.Lock()
	defer s.mu.Unlock()
	return emitted{vault: v.ID, texts: s.textCalls, plans: s.planCalls, timers: len(s.timers)}
}

func (s *steps) waitForTimer(t *testing.T, n int) chan time.Time {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		s.mu.Lock()
		if len(s.timers) >= n {
			timer := s.timers[n-1]
			s.mu.Unlock()
			return timer
		}
		s.mu.Unlock()
		select {
		case <-deadline:
			t.Fatalf("the pass never started quiet timer %d", n)
		case <-time.After(time.Millisecond):
		}
	}
}

// session is one vault indexed by a worker over steps, with the listener of index changes.
type session struct {
	worker *Worker
	run    *Session
	steps  *steps
	events *recorder
	v      domain.Vault

	mu    sync.Mutex
	heard []emitted
}

func newSession(withSource bool) *session {
	s := &session{steps: &steps{}, events: newRecorder(), v: domain.Vault{ID: "01VAULT", Name: "Field notes"}}
	var source searchsourceport.Source
	if withSource {
		source = sourcefake.Wrap(&fakeEmbedder{dims: 64})
	}
	s.worker = NewWorker(Config{
		Reading: func(context.Context, searchsourceport.Source, domain.Vault, bool) (ReadWholeVault, error) {
			return NewReadWholeVault(s.steps, s.steps, s.steps), nil
		},
		Source:       source,
		VectorCounts: s.steps.getVectorCounts,
		IndexChanged: func(_ context.Context, v domain.Vault) {
			at := s.steps.snapshot(v)
			s.mu.Lock()
			defer s.mu.Unlock()
			s.heard = append(s.heard, at)
		},
		Events: s.events,
		Quiet:  QuietPeriod,
		After:  s.steps.startTimer,
	})
	s.run = s.worker.NewSession()
	return s
}

func (s *session) start(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-ended
	})
	go func() {
		defer close(ended)
		s.run.Run(ctx, s.v, false, s.readNotes)
	}()
}

func (s *session) readNotes(ctx context.Context, _ func()) (vaultreading.ScanResult, error) {
	return s.steps.ScanWithProgress(ctx, s.v, nil)
}

func (s *session) getHeard() []emitted {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.heard)
}

func (s *session) waitForHeard(t *testing.T, n int) []emitted {
	t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		if heard := s.getHeard(); len(heard) >= n {
			return heard
		}
		select {
		case <-deadline:
			t.Fatalf("the listener received %v, want %d events", s.getHeard(), n)
		case <-time.After(time.Millisecond):
		}
	}
}

func (s *session) settle(t *testing.T, texts, plans int) {
	t.Helper()
	// Nothing marks the end of a step that sends no event, so the wait ends with a sleep: a wake sent
	// before the step ends would hold the event back for a reason other than the one a test checks.
	deadline := time.After(10 * time.Second)
	for {
		at := s.steps.snapshot(s.v)
		if at.texts >= texts && at.plans >= plans {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("the pass reached %+v, want %d text steps and %d vector plans", at, texts, plans)
		case <-time.After(time.Millisecond):
		}
	}
	time.Sleep(50 * time.Millisecond)
}

func (s *session) cutOneSourceMore() {
	// Every step of the pass runs on one goroutine, so the event of an earlier step arrives before
	// the event of this text step.
	s.steps.queueTexts(vaultreading.IndexResult{Seen: 1, Extracted: 1})
	s.worker.WakeSources()
}

func (s *session) checkHeard(t *testing.T, want ...emitted) {
	t.Helper()
	heard := s.waitForHeard(t, len(want))
	if !slices.Equal(heard, want) {
		t.Errorf("the listener heard %+v, want %+v", heard, want)
	}
}

func (s *session) checkHeardNothing(t *testing.T) {
	t.Helper()
	if heard := s.getHeard(); len(heard) != 0 {
		t.Fatalf("the listener heard %+v, want nothing yet", heard)
	}
}

func TestPassThatOnlyRemovesChunksEmitsIndexChanged(t *testing.T) {
	s := newSession(true)
	s.steps.texts = []vaultreading.IndexResult{{Seen: 2, Unchanged: 1, Removed: 1}}
	s.steps.vectors = append(s.steps.vectors, func(context.Context, func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
		t.Error("vectors were made for a vault that only lost chunks")
		return vaultreading.EmbedResult{}, nil
	})
	s.start(t)

	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1, plans: 1}, emitted{vault: s.v.ID, texts: 2, plans: 2})
}

func TestNotesWakeWithoutSourceEmitsAfterQuietTimer(t *testing.T) {
	s := newSession(false)
	s.start(t)
	s.settle(t, 1, 0)

	// The note was saved or removed by the refresh that sent the wake.
	s.worker.WakeNotes()
	timer := s.steps.waitForTimer(t, 1)
	s.settle(t, 1, 0)
	s.checkHeardNothing(t)
	timer <- time.Time{}
	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1, timers: 1}, emitted{vault: s.v.ID, texts: 2, timers: 1})
}

func TestFirstWalkWithoutSourceOverShortNoteEmits(t *testing.T) {
	s := newSession(false)
	// A note shorter than the shortest chunk is saved with no chunk that needs a vector.
	s.steps.read = vaultreading.ScanResult{Notes: 1, Indexed: 1}
	s.start(t)

	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1}, emitted{vault: s.v.ID, texts: 2})
}

func TestUnchangedSessionEmitsNothing(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withSource bool
		plans      int
	}{
		{name: "without a search source"},
		{name: "with a search source whose vectors are all made", withSource: true, plans: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSession(tc.withSource)
			s.steps.read = vaultreading.ScanResult{Notes: 3, Unchanged: 3}
			s.steps.texts = []vaultreading.IndexResult{{Seen: 2, Unchanged: 2}}
			s.start(t)

			// The first pass of the window walked, cut and embedded nothing new.
			s.settle(t, 1, tc.plans)
			s.cutOneSourceMore()

			s.checkHeard(t, emitted{vault: s.v.ID, texts: 2, plans: 2 * tc.plans})
		})
	}
}

func TestWritingVectorsEmitsOnce(t *testing.T) {
	s := newSession(true)
	s.steps.isPending = true
	s.steps.vectors = append(s.steps.vectors, func(context.Context, func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
		s.steps.setPending(false)
		return vaultreading.EmbedResult{Embedded: 4}, nil
	})
	s.start(t)
	s.waitForHeard(t, 1)

	// A following step finds nothing new to cut or embed.
	s.worker.WakeSources()
	s.settle(t, 2, 2)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1, plans: 1}, emitted{vault: s.v.ID, texts: 3, plans: 3})
}

func failVectors(context.Context, func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
	return vaultreading.EmbedResult{}, errors.New("the model is not answering")
}

func TestFailedVectorStepWithChangedChunksEmits(t *testing.T) {
	s := newSession(true)
	s.steps.isPending = true
	s.steps.texts = []vaultreading.IndexResult{{Seen: 1, Extracted: 1}}
	s.steps.vectors = append(s.steps.vectors, failVectors, failVectors)
	s.start(t)

	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1, plans: 1}, emitted{vault: s.v.ID, texts: 2, plans: 2})
}

func TestFailedVectorStepWithNothingChangedEmitsNothing(t *testing.T) {
	s := newSession(true)
	s.steps.isPending = true
	s.steps.vectors = append(s.steps.vectors, failVectors, failVectors)
	s.start(t)

	s.settle(t, 1, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 2, plans: 2})
}

func TestVectorStepStoppedByWakeEmitsNothing(t *testing.T) {
	s := newSession(true)
	s.steps.isPending = true
	s.steps.texts = []vaultreading.IndexResult{{Seen: 1, Extracted: 1}}
	s.steps.vectors = append(s.steps.vectors,
		func(ctx context.Context, _ func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
			// A book is added while the vectors are made: the pass stands aside for it.
			s.worker.WakeSources()
			<-ctx.Done()
			return vaultreading.EmbedResult{Embedded: 2}, ctx.Err()
		},
		func(context.Context, func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
			s.steps.setPending(false)
			return vaultreading.EmbedResult{Embedded: 2}, nil
		},
	)
	s.start(t)

	// The step that follows the wake finishes the vectors and tells of the changes of both steps once.
	s.checkHeard(t, emitted{vault: s.v.ID, texts: 2, plans: 2})
}

func TestVectorStepCanceledByAPersonAfterWritingVectorsEmits(t *testing.T) {
	s := newSession(true)
	s.steps.isPending = true
	started := make(chan struct{})
	s.steps.vectors = append(s.steps.vectors,
		func(ctx context.Context, onProgress func(vaultreading.EmbedResult)) (vaultreading.EmbedResult, error) {
			onProgress(vaultreading.EmbedResult{Embedded: 2, CurrentSource: "books/tide.epub"})
			close(started)
			<-ctx.Done()
			return vaultreading.EmbedResult{Embedded: 2}, ctx.Err()
		},
	)
	s.start(t)

	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("the pass never made vectors")
	}
	if !s.events.tasks.Cancel(work.MakingVectors) {
		t.Fatal("the vector step is not cancellable")
	}
	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1, plans: 1}, emitted{vault: s.v.ID, texts: 2, plans: 2})
}

func TestCancelledSessionEmitsNothing(t *testing.T) {
	for _, withSource := range []bool{false, true} {
		s := newSession(withSource)
		s.steps.read = vaultreading.ScanResult{Notes: 1, Indexed: 1}
		s.steps.texts = []vaultreading.IndexResult{{Seen: 1, Extracted: 1}}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		s.run.Run(ctx, s.v, false, s.readNotes)

		if heard := s.getHeard(); len(heard) != 0 {
			t.Errorf("with a source %v, the listener heard %+v from a canceled session", withSource, heard)
		}
	}
}

func TestStepThatEndsWhileTheQuietTimerRunsEmitsAtTheTimer(t *testing.T) {
	s := newSession(true)
	s.start(t)
	s.settle(t, 1, 1)

	s.worker.WakeNotes()
	timer := s.steps.waitForTimer(t, 1)
	// The book is cut while the run of writes has not been quiet long enough.
	s.cutOneSourceMore()
	s.settle(t, 2, 2)
	s.checkHeardNothing(t)
	timer <- time.Time{}

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 2, plans: 3, timers: 1})
}

func TestIndexingRecognizedPagesEmits(t *testing.T) {
	s := newSession(false)
	s.steps.cuts = []vaultreading.IndexResult{{Seen: 1, Extracted: 1}}
	s.start(t)
	s.settle(t, 1, 0)

	s.run.RequestCut(s.v, "scans/page.png")
	s.waitForHeard(t, 1)
	s.cutOneSourceMore()

	s.checkHeard(t, emitted{vault: s.v.ID, texts: 1}, emitted{vault: s.v.ID, texts: 2})
}

func testAWorkerWithoutAListenerEmbedsAsBefore(t *testing.T, openReading openFunc) {
	t.Helper()
	made, db := openReading(t)
	model := &fakeEmbedder{dims: 64}
	fit(t, db, model)
	v := made.newVault(t, map[string]string{"Note.md": noteWith(getBeforeNames(), 200)})
	cut(t, made, db, v)
	events := newRecorder()

	newPass(made, db, events, model, model.Describe().Recipe(), v).embed(t.Context())

	if pos := findTask(t, events, work.MakingVectors); pos != nil {
		t.Errorf("a pass without a listener is still being done: %+v", *pos)
	}
}
