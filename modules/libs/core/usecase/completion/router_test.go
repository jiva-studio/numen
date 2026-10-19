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

// choices holds the choice of each plugin's setting.
type choices map[string]port.CompletionChoice

func (c choices) GetCompletionChoice(pluginID string) port.CompletionChoice {
	return c[pluginID]
}

func newRouter(t *testing.T, config completion.Config, chosen choices, fake *agents) *completion.Router {
	t.Helper()
	router, err := completion.NewRouter(func() completion.Config { return config }, chosen, fake)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

// A router is made with what it needs or not at all.
func TestARouterIsRefusedWhatItCannotWorkWithout(t *testing.T) {
	config := func() completion.Config { return completion.Config{} }
	if _, err := completion.NewRouter(nil, choices{}, &agents{}); err == nil {
		t.Error("a router without settings was made")
	}
	if _, err := completion.NewRouter(config, nil, &agents{}); err == nil {
		t.Error("a router without choices was made")
	}
	if _, err := completion.NewRouter(config, choices{}, nil); err == nil {
		t.Error("a router without agents was made")
	}
}

// A plugin naming no agent is answered by the default agent and model, and the
// agent is closed once it has answered.
func TestAPluginNamingNoAgentIsAnsweredByTheDefault(t *testing.T) {
	fake := &agents{}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d", Model: "model-d"}}, choices{}, fake)

	got, err := router.Complete(t.Context(), "plugin.a", port.Question{Purpose: "proofreading"})
	if err != nil || got != "answer to proofreading" {
		t.Fatalf("answered %q, %v", got, err)
	}
	if len(fake.opened) != 1 || fake.opened[0] != "agent-d" || fake.models[0] != "model-d" {
		t.Errorf("opened %v with %v, want agent-d with model-d", fake.opened, fake.models)
	}
	if fake.closed.Load() != 1 {
		t.Errorf("%d agents closed, want 1", fake.closed.Load())
	}
}

// An empty default opens the agent by the empty name, which is the provider
// with the lowest id, and leaves the model to it.
func TestAnEmptyDefaultLeavesTheAgentAndTheModelToTheProviders(t *testing.T) {
	fake := &agents{}
	router := newRouter(t, completion.Config{}, choices{}, fake)
	if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
		t.Fatal(err)
	}
	if fake.opened[0] != "" || fake.models[0] != "" {
		t.Errorf("opened %q with model %q, want both empty", fake.opened[0], fake.models[0])
	}
}

// A plugin whose setting names an agent is answered by that agent with the
// model the setting names, and the default model goes with the default alone.
func TestAPluginsOwnChoiceStandsOverTheDefault(t *testing.T) {
	fake := &agents{}
	router := newRouter(t,
		completion.Config{Default: completion.Route{Agent: "agent-d", Model: "model-d"}},
		choices{
			"plugin.a": {Agent: "agent-a", Model: "model-a"},
			"plugin.b": {Agent: "agent-b"},
			"plugin.c": {Model: "ignored"},
		},
		fake)

	for _, one := range []string{"plugin.a", "plugin.b", "plugin.c", "plugin.d"} {
		if _, err := router.Complete(t.Context(), one, port.Question{}); err != nil {
			t.Fatal(err)
		}
	}
	wantAgents := []string{"agent-a", "agent-b", "agent-d", "agent-d"}
	wantModels := []string{"model-a", "", "model-d", "model-d"}
	for at := range wantAgents {
		if fake.opened[at] != wantAgents[at] || fake.models[at] != wantModels[at] {
			t.Errorf("question %d went to %q with %q, want %q with %q",
				at, fake.opened[at], fake.models[at], wantAgents[at], wantModels[at])
		}
	}
}

// The agent a plugin's setting names fails the question as it fails: the
// question is not taken to any other agent.
func TestAMissingAgentFailsTheQuestionWithNoOtherAgentAsked(t *testing.T) {
	fake := &agents{openErr: port.ErrNoRoute}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d"}},
		choices{"plugin.a": {Agent: "absent"}}, fake)

	for range 3 {
		if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); !errors.Is(err, port.ErrNoRoute) {
			t.Fatalf("error = %v, want port.ErrNoRoute", err)
		}
	}
	if len(fake.opened) != 0 {
		t.Errorf("opened %v", fake.opened)
	}
}

// What a question says about its purpose routes nothing: only the plugin's
// setting and the default do.
func TestAQuestionCannotChooseItsRoute(t *testing.T) {
	fake := &agents{}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d"}}, choices{}, fake)
	for _, purpose := range []string{"proofreading", "md.numen.agent-claude", "default"} {
		if _, err := router.Complete(t.Context(), "plugin.a", port.Question{Purpose: purpose}); err != nil {
			t.Fatal(err)
		}
	}
	for _, opened := range fake.opened {
		if opened != "agent-d" {
			t.Errorf("a question was taken to %q", opened)
		}
	}
}

// The settings and the choices are read at every question.
func TestAChoiceEditedInTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	fake := &agents{}
	config := completion.Config{Default: completion.Route{Agent: "early"}}
	chosen := choices{}
	router, err := completion.NewRouter(func() completion.Config { return config }, chosen, fake)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
		t.Fatal(err)
	}
	config = completion.Config{Default: completion.Route{Agent: "late"}}
	if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
		t.Fatal(err)
	}
	chosen["plugin.a"] = port.CompletionChoice{Agent: "own"}
	if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"early", "late", "own"}; fake.opened[0] != want[0] || fake.opened[1] != want[1] || fake.opened[2] != want[2] {
		t.Errorf("opened %v, want %v", fake.opened, want)
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

// No more questions answer at once than the limit.
func TestAnAgentAnswersNoMoreQuestionsAtOnceThanTheLimit(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{Default: completion.Route{Concurrency: 2}}, choices{}, &agents{answer: block.answer})

	const asked = 5
	var wg sync.WaitGroup
	for range asked {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
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

// Two agents do not wait for one another.
func TestTwoAgentsDoNotShareTheirPlaces(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{},
		choices{"plugin.a": {Agent: "a"}, "plugin.b": {Agent: "b"}}, &agents{answer: block.answer})

	var wg sync.WaitGroup
	for _, pluginID := range []string{"plugin.a", "plugin.b"} {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), pluginID, port.Question{}); err != nil {
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
	router := newRouter(t, completion.Config{Default: completion.Route{Concurrency: 1}}, choices{}, &agents{answer: block.answer})

	answering, stop := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() {
		_, err := router.Complete(answering, "plugin.a", port.Question{})
		first <- err
	}()
	<-block.started

	waiting, giveUp := context.WithCancel(t.Context())
	second := make(chan error, 1)
	go func() {
		_, err := router.Complete(waiting, "plugin.a", port.Question{})
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
	if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
		t.Errorf("the place was not given back: %v", err)
	}
}

// A question whose context is already cancelled opens no agent.
func TestAQuestionCancelledBeforeItStartsOpensNoAgent(t *testing.T) {
	fake := &agents{}
	router := newRouter(t, completion.Config{}, choices{}, fake)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := router.Complete(ctx, "plugin.a", port.Question{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(fake.opened) != 0 {
		t.Errorf("opened %v", fake.opened)
	}
}

// A limit changed in the settings applies to the questions that come after.
func TestAChangedLimitAppliesToTheNextQuestions(t *testing.T) {
	block := newBlocker()
	config := completion.Config{Default: completion.Route{Concurrency: 1}}
	router, err := completion.NewRouter(func() completion.Config { return config }, choices{}, &agents{answer: block.answer})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	// One question holds the only place under the limit of one.
	wg.Go(func() {
		if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
			t.Error(err)
		}
	})
	<-block.started

	config = completion.Config{Default: completion.Route{Concurrency: 3}}
	for range 3 {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), "plugin.a", port.Question{}); err != nil {
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
