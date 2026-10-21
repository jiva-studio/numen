package embedding_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// gated is a model that runs passes calls at once. Every call counts itself in,
// and waits on release for the text "hold"; any other text returns at once.
type gated struct {
	model   port.EmbeddingModel
	passes  int
	release chan struct{}
	running atomic.Int32
	peak    atomic.Int32
	entered chan struct{}
}

func newGated(passes int) *gated {
	return &gated{
		model: is("gated"), passes: passes,
		release: make(chan struct{}), entered: make(chan struct{}, 64),
	}
}

func (g *gated) GetModel() port.EmbeddingModel { return g.model }
func (g *gated) Concurrency() int              { return g.passes }
func (g *gated) Close() error                  { return nil }

func (g *gated) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	now := g.running.Add(1)
	defer g.running.Add(-1)
	for {
		observed := g.peak.Load()
		if now <= observed || g.peak.CompareAndSwap(observed, now) {
			break
		}
	}
	g.entered <- struct{}{}
	if texts[0] == "hold" {
		select {
		case <-g.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return [][]float32{{1, 0, 0}}, nil
}

func openGated(t *testing.T, passes int) (filling, asking port.Embedder, model *gated) {
	t.Helper()
	model = newGated(passes)
	filling, asking, closer := embedding.Open(t.Context(), run.NewTasks(), embedding.NewReached(model.model, "gated", model), embedding.Provider{})
	t.Cleanup(func() { _ = closer() })
	return filling, asking, model
}

func holdOnIndexSide(ctx context.Context, filling port.Embedder, calls int) (*sync.WaitGroup, chan error) {
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := filling.Embed(ctx, []string{"hold"})
			errs <- err
		}()
	}
	return &wg, errs
}

func waitEntered(t *testing.T, m *gated, n int) {
	t.Helper()
	for range n {
		select {
		case <-m.entered:
		case <-time.After(5 * time.Second):
			t.Fatal("a call never reached the model")
		}
	}
}

func TestAQuestionIsAnsweredWhileIndexingContainsEveryIndexSlot(t *testing.T) {
	for _, passes := range []int{2, 3, 8} {
		filling, asking, model := openGated(t, passes)
		ctx, cancel := context.WithCancel(t.Context())
		// More callers than slots: the extra ones wait on the lane.
		wg, _ := holdOnIndexSide(ctx, filling, passes*2)
		waitEntered(t, model, passes-1)

		reply := make(chan error, 1)
		go func() {
			_, err := asking.Embed(t.Context(), []string{"question"})
			reply <- err
		}()
		select {
		case err := <-reply:
			if err != nil {
				t.Errorf("%d passes: %v", passes, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("%d passes: a question waited behind indexing", passes)
		}
		if got := model.running.Load(); got != int32(passes-1) {
			t.Errorf("%d passes: %d calls held, want %d", passes, got, passes-1)
		}
		cancel()
		wg.Wait()
	}
}

func TestTheIndexSideNeverHasMoreCallsInFlightThanItsLimit(t *testing.T) {
	filling, _, model := openGated(t, 4)
	if got := filling.(interface{ Concurrency() int }).Concurrency(); got != 3 {
		t.Fatalf("the index side reports %d, want 3", got)
	}
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := filling.Embed(t.Context(), []string{"quick"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := model.peak.Load(); got > 3 {
		t.Errorf("%d index calls ran at once", got)
	}
}

func TestAModelOfOnePassLetsAQuestionThroughAndContainsOneIndexCall(t *testing.T) {
	filling, asking, model := openGated(t, 1)
	if got := filling.(interface{ Concurrency() int }).Concurrency(); got != 1 {
		t.Fatalf("the index side reports %d, want 1", got)
	}
	ctx, cancel := context.WithCancel(t.Context())
	wg, _ := holdOnIndexSide(ctx, filling, 3)
	waitEntered(t, model, 1)

	if _, err := asking.Embed(t.Context(), []string{"question"}); err != nil {
		t.Fatal(err)
	}
	if got := model.running.Load(); got != 1 {
		t.Errorf("%d index calls held, want 1", got)
	}
	cancel()
	wg.Wait()
}

func TestACancelledIndexCallFreesItsSlot(t *testing.T) {
	filling, _, model := openGated(t, 2)
	ctx, cancel := context.WithCancel(t.Context())
	wg, errs := holdOnIndexSide(ctx, filling, 3)
	waitEntered(t, model, 1)
	cancel()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err == nil {
			t.Error("a held call returned without an error")
		}
	}

	if _, err := filling.Embed(t.Context(), []string{"quick"}); err != nil {
		t.Fatalf("the slot was not freed: %v", err)
	}
}

func TestAWaitForASlotEndsWithItsContext(t *testing.T) {
	filling, _, model := openGated(t, 2)
	_, _ = holdOnIndexSide(t.Context(), filling, 1)
	waitEntered(t, model, 1)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, err := filling.Embed(ctx, []string{"quick"}); err != context.DeadlineExceeded {
		t.Errorf("got %v", err)
	}
	close(model.release)
}

func TestTheLaneStartsNoGoroutineThatOutlivesACall(t *testing.T) {
	before := runtime.NumGoroutine()
	filling, asking, model := openGated(t, 3)
	close(model.release)
	for range 20 {
		if _, err := filling.Embed(t.Context(), []string{"hold"}); err != nil {
			t.Fatal(err)
		}
		if _, err := asking.Embed(t.Context(), []string{"question"}); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines went from %d to %d", before, after)
	}
}

func TestASeparateQuestionProviderPreservesTheIndexSideUnlimited(t *testing.T) {
	first, second := newGated(4), newGated(4)
	second.model = first.model
	filling, _, closer := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewReached(first.model, "first", first), embedding.NewReached(second.model, "second", second))
	t.Cleanup(func() { _ = closer() })
	if got := filling.(interface{ Concurrency() int }).Concurrency(); got != 4 {
		t.Errorf("the index side reports %d, want all 4", got)
	}
}

func TestTheIndexSideIsSizedByTheModelThatArrives(t *testing.T) {
	model := newGated(4)
	arrive := make(chan struct{})
	opening := func(ctx context.Context, _ func(done, total int64)) (port.Embedder, error) {
		select {
		case <-arrive:
			return model, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	filling, _, closer := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewFetched(model.model, "gated", opening), embedding.Provider{})
	t.Cleanup(func() { _ = closer() })

	first := make(chan error, 1)
	go func() {
		_, err := filling.Embed(t.Context(), []string{"quick"})
		first <- err
	}()
	time.Sleep(100 * time.Millisecond)
	close(arrive)
	select {
	case err := <-first:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a call made before the model arrived never returned")
	}
	waitEntered(t, model, 1)

	ctx, cancel := context.WithCancel(t.Context())
	wg, _ := holdOnIndexSide(ctx, filling, 6)
	waitEntered(t, model, 3)
	if got := model.running.Load(); got != 3 {
		t.Errorf("%d index calls held, want the 3 a model of 4 passes allows", got)
	}
	cancel()
	wg.Wait()
}

func TestTheIndexSideLetsGoOfTheModelAndReturnsWhatClosingSaid(t *testing.T) {
	target := newSaying("closing", 1, 0, 0)
	filling, _, letGo := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewReached(target.model, "closing", closing{saying: target, why: errModelGone}), embedding.Provider{})
	_ = letGo

	if err := filling.Close(); !errors.Is(err, errModelGone) {
		t.Errorf("close = %v, want what closing the model said", err)
	}
	if target.closed.Load() != 1 {
		t.Error("the model was not let go of")
	}
}

func TestTheIndexSideReturnsWhyTheModelNeverArrives(t *testing.T) {
	filling, _, letGo := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewFetched(is("never"), "never", failWith(errModelGone)), embedding.Provider{})
	defer func() { _ = letGo() }()

	waiter, ok := filling.(interface{ Wait(context.Context) error })
	if !ok {
		t.Fatal("the index side cannot be waited on")
	}
	if err := waiter.Wait(t.Context()); !errors.Is(err, errModelGone) {
		t.Errorf("wait = %v, want why the model never arrived", err)
	}
}
