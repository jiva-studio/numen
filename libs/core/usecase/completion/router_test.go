package completion_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/completion"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
	"github.com/jiva-studio/numen/libs/core/port/completionport"
)

// defaultProgram is the program the programs name for an empty name.
const defaultProgram = "installed-default"

// programs finds a program per name and records the tasks it was asked to run.
type programs struct {
	mu        sync.Mutex
	requested []string
	tasks     []agentport.Task
	getErr    error
	models    []agentport.AgentModel
	startErr  error
	answer    func(ctx context.Context, task agentport.Task) (string, error)
}

func (p *programs) GetProgram(_ context.Context, name string) (agentport.AgentProgram, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requested = append(p.requested, name)
	if p.getErr != nil {
		return agentport.AgentProgram{}, p.getErr
	}
	if name == "" {
		name = defaultProgram
	}
	return agentport.AgentProgram{Name: name, IsInstalled: true, Models: p.models}, nil
}

func (p *programs) StartRun(ctx context.Context, task agentport.Task) (agentport.Run, error) {
	p.mu.Lock()
	p.tasks = append(p.tasks, task)
	p.mu.Unlock()
	if p.startErr != nil {
		return nil, p.startErr
	}
	text := "answer to " + task.Question
	if p.answer != nil {
		var err error
		if text, err = p.answer(ctx, task); err != nil {
			return nil, err
		}
	}
	steps := make(chan agentport.Step, 2)
	steps <- agentport.Step{Kind: domain.StepSay, Text: text}
	steps <- agentport.Step{Kind: domain.StepStop}
	close(steps)
	return &run{steps: steps}, nil
}

type run struct{ steps chan agentport.Step }

func (r *run) GetSteps() <-chan agentport.Step { return r.steps }
func (r *run) Stop() error                     { return nil }

// choices holds the choice of each plugin's setting.
type choices map[string]completionport.CompletionChoice

func (c choices) GetCompletionChoice(pluginID string) completionport.CompletionChoice {
	return c[pluginID]
}

// ask is a question with one message, the way a plugin asks it.
func ask(text string) completionport.Question {
	return completionport.Question{Messages: []completionport.Message{{Role: "user", Text: text}}}
}

func newRouter(t *testing.T, config completion.Config, choice choices, fake *programs) *completion.Router {
	t.Helper()
	router, err := completion.NewRouter(func() completion.Config { return config }, choice, fake)
	if err != nil {
		t.Fatal(err)
	}
	return router
}

// A router is made with what it needs or not at all.
func TestARouterIsRefusedWhatItCannotWorkWithout(t *testing.T) {
	config := func() completion.Config { return completion.Config{} }
	if _, err := completion.NewRouter(nil, choices{}, &programs{}); err == nil {
		t.Error("a router without settings was made")
	}
	if _, err := completion.NewRouter(config, nil, &programs{}); err == nil {
		t.Error("a router without choices was made")
	}
	if _, err := completion.NewRouter(config, choices{}, nil); err == nil {
		t.Error("a router without programs was made")
	}
}

// A plugin naming no agent is answered by the default agent and model, through
// a task that names no tool and no conversation.
func TestAPluginNamingNoAgentIsAnsweredByTheDefault(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d", Model: "model-d"}}, choices{}, fake)

	got, err := router.Complete(t.Context(), "plugin.a", ask("why?"))
	if err != nil || got != "answer to why?" {
		t.Fatalf("answered %q, %v", got, err)
	}
	if len(fake.tasks) != 1 {
		t.Fatalf("%d tasks started, want 1", len(fake.tasks))
	}
	task := fake.tasks[0]
	if task.Agent != "agent-d" || task.Model != "model-d" || len(task.Tools) != 0 ||
		len(task.ToolDeclarations) != 0 || task.Conversation != "" || task.Endpoint != (agentport.ToolEndpoint{}) {
		t.Errorf("the task is %+v, want agent-d with model-d and no tool or conversation", task)
	}
}

// An empty default names the program by the empty name, which the programs
// resolve, and leaves the model to the program.
func TestAnEmptyDefaultPreservesTheAgentAndTheModelToThePrograms(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{}, choices{}, fake)
	if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
		t.Fatal(err)
	}
	if fake.requested[0] != "" || fake.tasks[0].Agent != defaultProgram || fake.tasks[0].Model != "" {
		t.Errorf("asked %q, ran on %q with model %q", fake.requested[0], fake.tasks[0].Agent, fake.tasks[0].Model)
	}
}

// A plugin whose setting names an agent is answered by that agent with the
// model the setting names, and the default model goes with the default alone.
func TestAPluginsOwnChoiceExistsOverTheDefault(t *testing.T) {
	fake := &programs{}
	router := newRouter(t,
		completion.Config{Default: completion.Route{Agent: "agent-d", Model: "model-d"}},
		choices{
			"plugin.a": {Agent: "agent-a", Model: "model-a"},
			"plugin.b": {Agent: "agent-b"},
			"plugin.c": {Model: "ignored"},
		},
		fake)

	for _, one := range []string{"plugin.a", "plugin.b", "plugin.c", "plugin.d"} {
		if _, err := router.Complete(t.Context(), one, ask("why?")); err != nil {
			t.Fatal(err)
		}
	}
	wantAgents := []string{"agent-a", "agent-b", "agent-d", "agent-d"}
	wantModels := []string{"model-a", "", "model-d", "model-d"}
	for item := range wantAgents {
		if fake.tasks[item].Agent != wantAgents[item] || fake.tasks[item].Model != wantModels[item] {
			t.Errorf("question %d went to %q with %q, want %q with %q",
				item, fake.tasks[item].Agent, fake.tasks[item].Model, wantAgents[item], wantModels[item])
		}
	}
}

// The agent a plugin's setting names fails the question as it fails: the
// question is not taken to any other agent.
func TestAMissingAgentFailsTheQuestionWithNoOtherAgentAsked(t *testing.T) {
	fake := &programs{getErr: completionport.ErrNoRoute}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d"}},
		choices{"plugin.a": {Agent: "absent"}}, fake)

	for range 3 {
		if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); !errors.Is(err, completionport.ErrNoRoute) {
			t.Fatalf("error = %v, want port.ErrNoRoute", err)
		}
	}
	if len(fake.tasks) != 0 {
		t.Errorf("started %v", fake.tasks)
	}
	for _, request := range fake.requested {
		if request != "absent" {
			t.Errorf("asked for %q", request)
		}
	}
}

// An agent that cannot be reached or says nothing fails the question as an unavailability.
func TestAnAgentThatCannotAnswerIsUnavailable(t *testing.T) {
	for name, fake := range map[string]*programs{
		"cannot start": {startErr: errors.New("no plugin runs it")},
		"says nothing": {answer: func(context.Context, agentport.Task) (string, error) { return " ", nil }},
	} {
		router := newRouter(t, completion.Config{}, choices{}, fake)
		if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); !errors.Is(err, completionport.ErrUnavailable) {
			t.Errorf("%s: error = %v, want port.ErrUnavailable", name, err)
		}
	}
}

// What a question says about its purpose routes nothing: only the plugin's
// setting and the default do.
func TestAQuestionCannotChooseItsRoute(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{Default: completion.Route{Agent: "agent-d"}}, choices{}, fake)
	for _, purpose := range []string{"proofreading", "claude", "default"} {
		question := ask("why?")
		question.Purpose = purpose
		if _, err := router.Complete(t.Context(), "plugin.a", question); err != nil {
			t.Fatal(err)
		}
	}
	for _, task := range fake.tasks {
		if task.Agent != "agent-d" {
			t.Errorf("a question was taken to %q", task.Agent)
		}
	}
}

// The system messages are the instructions of the task and the others its
// prompt: as written when there is one, with their roles when there are several.
func TestTheQuestionIsTheTasksPromptAndInstructions(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{}, choices{}, fake)

	if _, err := router.Complete(t.Context(), "plugin.a", completionport.Question{Messages: []completionport.Message{
		{Role: "system", Text: "be brief"}, {Role: "system", Text: "be kind"}, {Text: "why?"},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := router.Complete(t.Context(), "plugin.a", completionport.Question{Messages: []completionport.Message{
		{Role: "user", Text: "why?"}, {Role: "assistant", Text: "because"}, {Role: "user", Text: "and?"},
	}}); err != nil {
		t.Fatal(err)
	}

	if got := fake.tasks[0]; got.Question != "why?" || got.Instructions != "be brief\n\nbe kind" {
		t.Errorf("the first task is %+v", got)
	}
	if got := fake.tasks[1]; got.Question != "User:\nwhy?\n\nAssistant:\nbecause\n\nUser:\nand?" || got.Instructions != "" {
		t.Errorf("the second task is %+v", got)
	}
}

func TestAQuestionNoAgentCanAnswerIsRefused(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{}, choices{}, fake)

	for name, question := range map[string]completionport.Question{
		"empty":        {},
		"only system":  {Messages: []completionport.Message{{Role: "system", Text: "be brief"}}},
		"unknown role": {Messages: []completionport.Message{{Role: "tool", Text: "why?"}}},
	} {
		if _, err := router.Complete(t.Context(), "plugin.a", question); !errors.Is(err, completionport.ErrInvalidQuestion) {
			t.Errorf("%s: error = %v, want port.ErrInvalidQuestion", name, err)
		}
	}
	if len(fake.requested) != 0 || len(fake.tasks) != 0 {
		t.Errorf("asked %v, started %v", fake.requested, fake.tasks)
	}
}

// The settings and the choices are read at every question.
func TestAChoiceEditedInTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	fake := &programs{}
	config := completion.Config{Default: completion.Route{Agent: "early"}}
	selected := choices{}
	router, err := completion.NewRouter(func() completion.Config { return config }, selected, fake)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
		t.Fatal(err)
	}
	config = completion.Config{Default: completion.Route{Agent: "late"}}
	if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
		t.Fatal(err)
	}
	selected["plugin.a"] = completionport.CompletionChoice{Agent: "own"}
	if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
		t.Fatal(err)
	}
	if want := []string{"early", "late", "own"}; fake.tasks[0].Agent != want[0] || fake.tasks[1].Agent != want[1] || fake.tasks[2].Agent != want[2] {
		t.Errorf("went to %v, want %v", []string{fake.tasks[0].Agent, fake.tasks[1].Agent, fake.tasks[2].Agent}, want)
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

func (b *blocker) answer(ctx context.Context, _ agentport.Task) (string, error) {
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
	router := newRouter(t, completion.Config{Default: completion.Route{Concurrency: 2}}, choices{}, &programs{answer: block.answer})

	const request = 5
	var wg sync.WaitGroup
	for range request {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
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
		choices{"plugin.a": {Agent: "a"}, "plugin.b": {Agent: "b"}}, &programs{answer: block.answer})

	var wg sync.WaitGroup
	for _, pluginID := range []string{"plugin.a", "plugin.b"} {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), pluginID, ask("why?")); err != nil {
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
func TestACancelledQuestionPreservesTheQueueAndStopsTheAnswer(t *testing.T) {
	block := newBlocker()
	router := newRouter(t, completion.Config{Default: completion.Route{Concurrency: 1}}, choices{}, &programs{answer: block.answer})

	answering, stop := context.WithCancel(t.Context())
	first := make(chan error, 1)
	go func() {
		_, err := router.Complete(answering, "plugin.a", ask("why?"))
		first <- err
	}()
	<-block.started

	waiting, giveUp := context.WithCancel(t.Context())
	second := make(chan error, 1)
	go func() {
		_, err := router.Complete(waiting, "plugin.a", ask("why?"))
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
	if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
		t.Errorf("the place was not given back: %v", err)
	}
}

// A question whose context is already cancelled starts no task.
func TestAQuestionCancelledBeforeItStartsStartsNoTask(t *testing.T) {
	fake := &programs{}
	router := newRouter(t, completion.Config{}, choices{}, fake)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := router.Complete(ctx, "plugin.a", ask("why?")); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(fake.tasks) != 0 {
		t.Errorf("started %v", fake.tasks)
	}
}

// A limit changed in the settings applies to the questions that come after.
func TestAChangedLimitAppliesToTheNextQuestions(t *testing.T) {
	block := newBlocker()
	config := completion.Config{Default: completion.Route{Concurrency: 1}}
	router, err := completion.NewRouter(func() completion.Config { return config }, choices{}, &programs{answer: block.answer})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	// One question holds the only place under the limit of one.
	wg.Go(func() {
		if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
			t.Error(err)
		}
	})
	<-block.started

	config = completion.Config{Default: completion.Route{Concurrency: 3}}
	for range 3 {
		wg.Go(func() {
			if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
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

// A model the program does not list is replaced by the one it answers with unasked, else by its first.
func TestAModelTheProgramDoesNotListIsReplaced(t *testing.T) {
	for name, test := range map[string]struct {
		models []agentport.AgentModel
		want   string
	}{
		"its own default": {[]agentport.AgentModel{{Name: "m-1"}, {Name: "m-2", IsDefault: true}}, "m-2"},
		"its first":       {[]agentport.AgentModel{{Name: "m-1"}, {Name: "m-2"}}, "m-1"},
	} {
		fake := &programs{models: test.models}
		router := newRouter(t, completion.Config{}, choices{"plugin.a": {Agent: "agent-a", Model: "m-old"}}, fake)

		if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := fake.tasks[0].Model; got != test.want {
			t.Errorf("%s: asked model %q, want %q", name, got, test.want)
		}
	}
}

// A model the program lists, and a program that lists none, leave the chosen model as it is.
func TestAModelTheProgramListsIsCached(t *testing.T) {
	for name, models := range map[string][]agentport.AgentModel{
		"listed": {{Name: "m-old"}, {Name: "m-2", IsDefault: true}},
		"none":   nil,
	} {
		fake := &programs{models: models}
		router := newRouter(t, completion.Config{}, choices{"plugin.a": {Agent: "agent-a", Model: "m-old"}}, fake)

		if _, err := router.Complete(t.Context(), "plugin.a", ask("why?")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := fake.tasks[0].Model; got != "m-old" {
			t.Errorf("%s: asked model %q, want m-old", name, got)
		}
	}
}

// A failure after a replacement states the model asked and the one asked instead.
func TestAFailureAfterAReplacementStatesIt(t *testing.T) {
	fake := &programs{models: []agentport.AgentModel{{Name: "m-1"}}, startErr: errors.New("the agent stopped: boom")}
	router := newRouter(t, completion.Config{}, choices{"plugin.a": {Agent: "agent-a", Model: "m-old"}}, fake)

	_, err := router.Complete(t.Context(), "plugin.a", ask("why?"))

	if !errors.Is(err, completionport.ErrUnavailable) {
		t.Fatalf("error = %v, want port.ErrUnavailable", err)
	}
	for _, want := range []string{"boom", `"m-old"`, "agent-a", `"m-1"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}
