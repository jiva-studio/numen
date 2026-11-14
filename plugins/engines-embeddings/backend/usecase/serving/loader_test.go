package serving

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

type fakeModel struct {
	concurrency int
	closeErr    error
	closed      chan struct{}
}

func (m *fakeModel) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (m *fakeModel) Concurrency() int { return m.concurrency }

func (m *fakeModel) Close() error {
	if m.closed != nil {
		close(m.closed)
	}
	return m.closeErr
}

type harness struct {
	lock      sync.Mutex
	loader    *modelLoader[string]
	nices     []int
	reports   []string
	opened    []ports.Embedder
	openModel func(ctx context.Context, name string) (ports.Embedder, error)
}

func newHarness(t *testing.T, openModel func(ctx context.Context, name string) (ports.Embedder, error)) *harness {
	t.Helper()
	h := &harness{openModel: openModel}
	h.loader = newModelLoader(&h.lock, loaderPorts[string]{
		Open: func(ctx context.Context, name string) (ports.Embedder, error) { return h.openModel(ctx, name) },
		LowerNice: func(nice int) error {
			h.nices = append(h.nices, nice)
			return nil
		},
		Report: func(what string, err error) { h.reports = append(h.reports, what+": "+err.Error()) },
		Opened: func(emb ports.Embedder) { h.opened = append(h.opened, emb) },
	})
	t.Cleanup(func() {
		if err := h.loader.Close(); err != nil {
			t.Error(err)
		}
	})
	return h
}

func start(name string, nice int) func() loadStart[string] {
	return func() loadStart[string] { return loadStart[string]{Spec: name, Nice: nice} }
}

func TestTheModelOpensOnceForCallsThatWaitTogether(t *testing.T) {
	var opens atomic.Int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		opens.Add(1)
		entered <- struct{}{}
		<-release
		return &fakeModel{concurrency: 3}, nil
	})

	results := make(chan ports.Embedder, 2)
	for range 2 {
		go func() {
			emb, err := h.loader.Get(t.Context(), start("a", 10))
			if err != nil {
				t.Error(err)
			}
			results <- emb
		}()
	}
	<-entered
	close(release)

	first, second := <-results, <-results
	if first != second || opens.Load() != 1 {
		t.Fatalf("opened %d times, models %v and %v", opens.Load(), first, second)
	}
	if len(h.nices) != 1 || h.nices[0] != 10 || len(h.opened) != 1 {
		t.Errorf("nices %v, opened %d", h.nices, len(h.opened))
	}
}

func TestACanceledCallDoesNotStopTheOpening(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	h := newHarness(t, func(ctx context.Context, _ string) (ports.Embedder, error) {
		close(entered)
		select {
		case <-release:
			return &fakeModel{}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() { _, err := h.loader.Get(ctx, start("a", 0)); ended <- err }()
	<-entered
	cancel()
	if err := <-ended; !errors.Is(err, context.Canceled) {
		t.Fatalf("a canceled call returned %v", err)
	}

	close(release)
	if emb, err := h.loader.Get(t.Context(), start("a", 0)); err != nil || emb == nil {
		t.Fatalf("the next call: %v, %v", emb, err)
	}
}

func TestAFailedOpeningReturnsItsErrorAndTheNextCallOpensAgain(t *testing.T) {
	boom := errors.New("boom")
	calls := 0
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		calls++
		if calls == 1 {
			return nil, boom
		}
		return &fakeModel{}, nil
	})

	emb, err := h.loader.Get(t.Context(), start("a", 0))
	if !errors.Is(err, boom) || emb != nil || h.loader.IsLoaded() {
		t.Fatalf("Get = %v, %v, loaded %v", emb, err, h.loader.IsLoaded())
	}
	if emb, err = h.loader.Get(t.Context(), start("a", 0)); err != nil || emb == nil {
		t.Fatalf("the next Get = %v, %v", emb, err)
	}
}

func TestAResetDuringTheOpeningClosesThatModelAndOpensAnother(t *testing.T) {
	firstClosed := make(chan struct{})
	entered := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		calls++
		if calls == 1 {
			close(entered)
			<-release
			return &fakeModel{concurrency: 3, closed: firstClosed}, nil
		}
		return &fakeModel{concurrency: 5}, nil
	})

	result := make(chan ports.Embedder, 1)
	go func() {
		emb, err := h.loader.Get(t.Context(), start("a", 0))
		if err != nil {
			t.Error(err)
		}
		result <- emb
	}()
	<-entered
	h.lock.Lock()
	h.loader.ResetLocked()
	h.lock.Unlock()
	close(release)

	emb := <-result
	<-firstClosed
	if emb.Concurrency() != 5 || len(h.opened) != 1 || h.opened[0] != emb {
		t.Fatalf("got a model of %d passes, opened %d", emb.Concurrency(), len(h.opened))
	}
}

func TestAResetCancelsTheOpeningAndClosesTheLoadedModel(t *testing.T) {
	closed := make(chan struct{})
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		return &fakeModel{closed: closed}, nil
	})
	if _, err := h.loader.Get(t.Context(), start("a", 0)); err != nil {
		t.Fatal(err)
	}

	h.lock.Lock()
	h.loader.ResetLocked()
	isLoaded := h.loader.IsLoaded()
	h.lock.Unlock()

	<-closed
	if isLoaded {
		t.Fatal("the model is still loaded after the reset")
	}
}

func TestCloseCancelsTheOpeningAndReturnsTheErrorOfTheLoadedModel(t *testing.T) {
	wantErr := errors.New("close failed")
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		return &fakeModel{closeErr: wantErr}, nil
	})
	if _, err := h.loader.Get(t.Context(), start("a", 0)); err != nil {
		t.Fatal(err)
	}

	if err := h.loader.Close(); !errors.Is(err, wantErr) {
		t.Fatalf("Close error = %v, want %v", err, wantErr)
	}
	if h.loader.IsLoaded() {
		t.Fatal("Close left the model loaded")
	}
}

func TestCloseEndsTheContextOfAnOpeningInProgress(t *testing.T) {
	entered := make(chan struct{})
	canceled := make(chan struct{})
	var enteredOnce, canceledOnce sync.Once
	h := newHarness(t, func(ctx context.Context, _ string) (ports.Embedder, error) {
		enteredOnce.Do(func() { close(entered) })
		<-ctx.Done()
		canceledOnce.Do(func() { close(canceled) })
		return nil, ctx.Err()
	})
	ended := make(chan error, 1)
	go func() { _, err := h.loader.Get(t.Context(), start("a", 0)); ended <- err }()
	<-entered

	if err := h.loader.Close(); err != nil {
		t.Fatal(err)
	}

	<-canceled
}

func TestAFailedPriorityChangeIsReportedAndTheModelStaysLoaded(t *testing.T) {
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) { return &fakeModel{}, nil })
	failure := errors.New("no permission")
	h.loader.ports.LowerNice = func(int) error { return failure }

	if _, err := h.loader.Get(t.Context(), start("a", 0)); err != nil {
		t.Fatal(err)
	}

	if len(h.reports) != 1 || h.reports[0] != "set the priority of the embeddings plugin: no permission" {
		t.Errorf("reports = %v", h.reports)
	}
}

func TestStartLockedOpensTheModelOnceAndAGetDuringTheLoadWaitsForIt(t *testing.T) {
	var opens atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	loaded := &fakeModel{}
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		if opens.Add(1) == 1 {
			close(entered)
		}
		<-release
		return loaded, nil
	})

	h.lock.Lock()
	h.loader.StartLocked(start("a", 0))
	h.loader.StartLocked(start("a", 0))
	h.lock.Unlock()
	<-entered

	got := make(chan ports.Embedder, 1)
	go func() {
		emb, err := h.loader.Get(t.Context(), start("a", 0))
		if err != nil {
			t.Error(err)
		}
		got <- emb
	}()
	close(release)

	if emb := <-got; emb != loaded || opens.Load() != 1 {
		t.Fatalf("Get = %v after %d opens, want the started model after one", emb, opens.Load())
	}
	h.lock.Lock()
	h.loader.StartLocked(start("a", 0))
	h.lock.Unlock()
	if opens.Load() != 1 {
		t.Errorf("a start with the model loaded opened it again: %d opens", opens.Load())
	}
}

func TestAFailedLoadStartedInTheBackgroundIsReported(t *testing.T) {
	h := newHarness(t, func(context.Context, string) (ports.Embedder, error) {
		return nil, errors.New("model file is truncated")
	})
	reported := make(chan string, 1)
	h.loader.ports.Report = func(what string, err error) { reported <- what + ": " + err.Error() }

	h.lock.Lock()
	h.loader.StartLocked(start("a", 0))
	h.lock.Unlock()

	if got := <-reported; got != "load the embedding model: opening onnx embedding engine: model file is truncated" {
		t.Errorf("reported %q", got)
	}
}
