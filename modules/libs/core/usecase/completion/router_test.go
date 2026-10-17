package completion_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/completion"
)

// agents opens an agent per name and records what it was asked.
type agents struct {
	mu      sync.Mutex
	opened  []string
	models  []string
	closed  atomic.Int32
	openErr error
	answer  func(ctx context.Context, model string, question port.Question) (string, error)
}

func (a *agents) OpenCompletionAgent(_ context.Context, agent string) (port.CompletionAgent, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.openErr != nil {
		return nil, a.openErr
	}
	a.opened = append(a.opened, agent)
	return &answerer{owner: a}, nil
}

type answerer struct{ owner *agents }

func (a *answerer) Complete(ctx context.Context, model string, question port.Question) (string, error) {
	a.owner.mu.Lock()
	a.owner.models = append(a.owner.models, model)
	a.owner.mu.Unlock()
	if a.owner.answer != nil {
		return a.owner.answer(ctx, model, question)
	}
	return "answer to " + question.Purpose, nil
}

func (a *answerer) Close() error {
	a.owner.closed.Add(1)
	return nil
}

func newRouter(t *testing.T, config completion.Config, held *agents) *completion.Router {
	t.Helper()
	router, err := completion.NewRouter(func() completion.Config { return config }, held)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

// A router is made with what it needs or not at all.
func TestARouterIsRefusedWhatItCannotWorkWithout(t *testing.T) {
	if _, err := completion.NewRouter(nil, &agents{}); err == nil {
		t.Error("a router without routes was made")
	}
	if _, err := completion.NewRouter(func() completion.Config { return completion.Config{} }, nil); err == nil {
		t.Error("a router without agents was made")
	}
}

// The question goes to the agent and the model of its route, and the agent is
// let go once it has answered.
func TestAQuestionIsAnsweredByTheAgentAndModelOfItsRoute(t *testing.T) {
	held := &agents{}
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{
		"proofreading": {Agent: "agent-a", Model: "model-a"},
		"default":      {Agent: "agent-b", Model: "model-b"},
	}}, held)

	got, err := router.Complete(t.Context(), port.Question{Purpose: "proofreading"})
	if err != nil || got != "answer to proofreading" {
		t.Fatalf("answered %q, %v", got, err)
	}
	if _, err := router.Complete(t.Context(), port.Question{Purpose: "other"}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"agent-a", "agent-b"}; held.opened[0] != want[0] || held.opened[1] != want[1] {
		t.Errorf("opened %v, want %v", held.opened, want)
	}
	if want := []string{"model-a", "model-b"}; held.models[0] != want[0] || held.models[1] != want[1] {
		t.Errorf("asked models %v, want %v", held.models, want)
	}
	if held.closed.Load() != 2 {
		t.Errorf("%d agents let go, want 2", held.closed.Load())
	}
}

// A purpose with no route and no default is answered by no one, and no agent
// is opened for it.
func TestAQuestionWithoutARouteFailsWithNoRoute(t *testing.T) {
	held := &agents{}
	router := newRouter(t, completion.Config{}, held)

	_, err := router.Complete(t.Context(), port.Question{Purpose: "summary"})
	if !errors.Is(err, port.ErrNoRoute) {
		t.Fatalf("error = %v, want port.ErrNoRoute", err)
	}
	if !errors.Is(err, port.ErrUnavailable) {
		t.Errorf("error = %v, want it to be port.ErrUnavailable as well", err)
	}
	if len(held.opened) != 0 {
		t.Errorf("opened %v for a question nobody answers", held.opened)
	}
}

// An agent that cannot be opened fails the question, and the route's place is
// given back.
func TestAnAgentThatCannotBeOpenedFailsTheQuestion(t *testing.T) {
	held := &agents{openErr: port.ErrUnavailable}
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{
		"default": {},
	}}, held)

	for range 3 {
		if _, err := router.Complete(t.Context(), port.Question{}); !errors.Is(err, port.ErrUnavailable) {
			t.Fatalf("error = %v, want port.ErrUnavailable", err)
		}
	}
}

// The routes are read at every question.
func TestARouteEditedInTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	held := &agents{}
	config := completion.Config{}
	router, err := completion.NewRouter(func() completion.Config { return config }, held)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Complete(t.Context(), port.Question{Purpose: "p"}); !errors.Is(err, port.ErrNoRoute) {
		t.Fatalf("error = %v, want port.ErrNoRoute", err)
	}
	config = completion.Config{Routes: map[string]completion.Route{"p": {Agent: "late"}}}
	if _, err := router.Complete(t.Context(), port.Question{Purpose: "p"}); err != nil {
		t.Fatal(err)
	}
	if held.opened[0] != "late" {
		t.Errorf("opened %v", held.opened)
	}
}

// blocker answers once released, and counts how many questions are answering at once.
type blocker struct {
	release chan struct{}
	started chan struct{}
	now     atomic.Int32
	most    atomic.Int32
}

func newBlocker() *blocker {
	return &blocker{release: make(chan struct{}), started: make(chan struct{}, 16)}
}

func (b *blocker) answer(ctx context.Context, _ string, _ port.Question) (string, error) {
	running := b.now.Add(1)
	defer b.now.Add(-1)
	for {
		most := b.most.Load()
		if running <= most || b.most.CompareAndSwap(most, running) {
			break
		}
	}
	b.started <- struct{}{}
	select {
	case <-b.release:
		return "done", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// No more questions answer at once than the route's limit.
func TestARouteAnswersNoMoreQuestionsAtOnceThanItsLimit(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{
		"default": {Concurrency: 2},
	}}, &agents{answer: block.answer})

	const asked = 5
	var wg sync.WaitGroup
	for range asked {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), port.Question{}); err != nil {
				t.Error(err)
			}
		})
	}
	<-block.started
	<-block.started
	if got := block.now.Load(); got != 2 {
		t.Errorf("%d questions answering, want 2", got)
	}
	close(block.release)
	wg.Wait()
	if got := block.most.Load(); got != 2 {
		t.Errorf("at most %d questions answered at once, want 2", got)
	}
}

// Two routes do not wait for one another.
func TestTwoRoutesDoNotShareTheirPlaces(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{
		"a": {}, "b": {},
	}}, &agents{answer: block.answer})

	var wg sync.WaitGroup
	for _, purpose := range []string{"a", "b"} {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), port.Question{Purpose: purpose}); err != nil {
				t.Error(err)
			}
		})
	}
	<-block.started
	<-block.started
	close(block.release)
	wg.Wait()
}

// A question waiting for a place ends with its context, and the question
// answering ends with it too.
func TestACancelledQuestionLeavesTheQueueAndStopsTheAnswer(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{
		"default": {Concurrency: 1},
	}}, &agents{answer: block.answer})

	answering, stop := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() {
		_, err := router.Complete(answering, port.Question{})
		first <- err
	}()
	<-block.started

	waiting, giveUp := context.WithCancel(t.Context())
	second := make(chan error, 1)
	go func() {
		_, err := router.Complete(waiting, port.Question{})
		second <- err
	}()
	giveUp()
	if err := <-second; !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled wait ended with %v", err)
	}

	stop()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled answer ended with %v", err)
	}

	// The place is back: a third question is answered.
	close(block.release)
	if _, err := router.Complete(t.Context(), port.Question{}); err != nil {
		t.Errorf("the place was not given back: %v", err)
	}
}

// A question whose context is already cancelled opens no agent.
func TestAQuestionCancelledBeforeItStartsOpensNoAgent(t *testing.T) {
	held := &agents{}
	router := newRouter(t, completion.Config{Routes: map[string]completion.Route{"default": {}}}, held)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := router.Complete(ctx, port.Question{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(held.opened) != 0 {
		t.Errorf("opened %v", held.opened)
	}
}

// A limit changed in the settings applies to the questions that come after.
func TestAChangedLimitAppliesToTheNextQuestions(t *testing.T) {
	block := newBlocker()
	config := completion.Config{Routes: map[string]completion.Route{"default": {Concurrency: 1}}}
	router, err := completion.NewRouter(func() completion.Config { return config }, &agents{answer: block.answer})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	// One question holds the only place under the limit of one.
	wg.Go(func() {
		if _, err := router.Complete(t.Context(), port.Question{}); err != nil {
			t.Error(err)
		}
	})
	<-block.started

	config = completion.Config{Routes: map[string]completion.Route{"default": {Concurrency: 3}}}
	for range 3 {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), port.Question{}); err != nil {
				t.Error(err)
			}
		})
	}
	for range 3 {
		<-block.started
	}
	close(block.release)
	wg.Wait()
}
