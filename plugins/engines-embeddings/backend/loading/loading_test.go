package loading

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type model struct {
	concurrency int
	closeErr    error
	closed      chan struct{}
}

func (m *model) Embed(context.Context, []string) ([][]float32, error) { return nil, nil }

func (m *model) Concurrency() int { return m.concurrency }

func (m *model) Close() error {
	if m.closed != nil {
		close(m.closed)
	}
	return m.closeErr
}

type harness struct {
	lock      sync.Mutex
	loader    *Loader[string]
	nices     []int
	reports   []string
	opened    []Embedder
	openModel func(ctx context.Context, name string) (Embedder, error)
}

func newHarness(t *testing.T, openModel func(ctx context.Context, name string) (Embedder, error)) *harness {
	t.Helper()
	h := &harness{openModel: openModel}
	h.loader = New(&h.lock, Ports[string]{
		Open: func(ctx context.Context, name string) (Embedder, error) { return h.openModel(ctx, name) },
		LowerNice: func(nice int) error {
			h.nices = append(h.nices, nice)
			return nil
		},
		Report: func(what string, err error) { h.reports = append(h.reports, what+": "+err.Error()) },
		Opened: func(emb Embedder) { h.opened = append(h.opened, emb) },
	})
	t.Cleanup(func() {
		if err := h.loader.Close(); err != nil {
			t.Error(err)
		}
	})
	return h
}

func start(name string, nice int) func() Start[string] {
	return func() Start[string] { return Start[string]{Spec: name, Nice: nice} }
}

func TestTheModelOpensOnceForCallsThatWaitTogether(t *testing.T) {
	var opens atomic.Int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	h := newHarness(t, func(context.Context, string) (Embedder, error) {
		opens.Add(1)
		entered <- struct{}{}
		<-release
		return &model{concurrency: 3}, nil
	})

	results := make(chan Embedder, 2)
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
	h := newHarness(t, func(ctx context.Context, _ string) (Embedder, error) {
		close(entered)
		select {
		case <-release:
			return &model{}, nil
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
	h := newHarness(t, func(context.Context, string) (Embedder, error) {
		calls++
		if calls == 1 {
			return nil, boom
		}
		return &model{}, nil
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
	h := newHarness(t, func(context.Context, string) (Embedder, error) {
		calls++
		if calls == 1 {
			close(entered)
			<-release
			return &model{concurrency: 3, closed: firstClosed}, nil
		}
		return &model{concurrency: 5}, nil
	})

	result := make(chan Embedder, 1)
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
	h := newHarness(t, func(context.Context, string) (Embedder, error) {
		return &model{closed: closed}, nil
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
	h := newHarness(t, func(context.Context, string) (Embedder, error) {
		return &model{closeErr: wantErr}, nil
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
	h := newHarness(t, func(ctx context.Context, _ string) (Embedder, error) {
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
	h := newHarness(t, func(context.Context, string) (Embedder, error) { return &model{}, nil })
	failure := errors.New("no permission")
	h.loader.ports.LowerNice = func(int) error { return failure }

	if _, err := h.loader.Get(t.Context(), start("a", 0)); err != nil {
		t.Fatal(err)
	}

	if len(h.reports) != 1 || h.reports[0] != "set the priority of the embeddings plugin: no permission" {
		t.Errorf("reports = %v", h.reports)
	}
}
