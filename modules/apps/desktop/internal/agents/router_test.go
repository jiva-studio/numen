//go:build !nomcp

package agents

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// fakeProvider is the plugins, running tasks that report the steps it is given.
type fakeProvider struct {
	programs []port.AgentProgram
	listErr  error
	steps    []port.Step
	startErr error

	mu    sync.Mutex
	tasks []port.Task
	ended []string
	run   *fakeRun
}

func (f *fakeProvider) StartRun(_ context.Context, task port.Task) (port.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks = append(f.tasks, task)
	if f.startErr != nil {
		return nil, f.startErr
	}
	steps := make(chan port.Step, len(f.steps))
	for _, step := range f.steps {
		steps <- step
	}
	close(steps)
	f.run = &fakeRun{steps: steps}
	return f.run, nil
}

func (f *fakeProvider) EndConversation(_ context.Context, conversation string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ended = append(f.ended, conversation)
	return nil
}

func (f *fakeProvider) GetPrograms(context.Context, map[string][]string) ([]port.AgentProgram, error) {
	return f.programs, f.listErr
}

// getTasks are the tasks the plugins were given.
func (f *fakeProvider) getTasks() []port.Task {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.tasks)
}

type fakeRun struct {
	steps   chan port.Step
	mu      sync.Mutex
	stopped bool
}

func (r *fakeRun) GetSteps() <-chan port.Step { return r.steps }

func (r *fakeRun) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	return nil
}

func (r *fakeRun) isStopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopped
}

var vocabulary = map[string]mcp.Tool{
	"note_read": {Title: "Read notes", About: "paths", Inside: "path", Kind: domain.StepRead},
	"note_edit": {Title: "Edit a note", About: "path", Kind: domain.StepEdit, Match: "match", Text: "text"},
}

// makeRouter is a router over the provider, for the settings given.
func makeRouter(t *testing.T, provider port.AgentProvider, agent settings.Agent, preview EditPreview) (*Router, *bytes.Buffer) {
	t.Helper()

	out := &bytes.Buffer{}
	opts := Options{
		Config: makeInstallation(t), Root: "/vault", Out: out, Provider: provider, Preview: preview,
	}
	opts.Config.Agent = agent
	router := NewRouter(opts, "http://127.0.0.1:7717/mcp", "secret", vocabulary)
	t.Cleanup(func() { _ = router.Close() })
	return router, out
}

func drain(run port.Run) []port.Step {
	var steps []port.Step
	for step := range run.GetSteps() {
		steps = append(steps, step)
	}
	return steps
}

// A task naming no program goes to the configured one, and one naming another goes there.
func TestATaskGoesToTheProgramItNamesOrTheConfiguredOne(t *testing.T) {
	provider := &fakeProvider{}
	router, out := makeRouter(t, provider, settings.Agent{Use: settings.UseProgram("codex")}, EditPreview{})

	for _, asked := range []string{"", "claude"} {
		run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: asked})
		if err != nil {
			t.Fatal(err)
		}
		drain(run)
	}

	tasks := provider.getTasks()
	if len(tasks) != 2 || tasks[0].Agent != "codex" || tasks[1].Agent != "claude" {
		t.Fatalf("the plugins were given %+v", tasks)
	}
	if !strings.Contains(out.String(), "dispatching task to codex") {
		t.Errorf("the router said %q", out.String())
	}
}

// An installation that leaves the choice open answers with the program the plugins mark as the
// default, else the first they list.
func TestTheDefaultProgramAnswersWhereTheSettingsLeaveTheChoice(t *testing.T) {
	for name, programs := range map[string][]port.AgentProgram{
		"one marked":  {{Name: "first"}, {Name: "marked", IsDefault: true}},
		"none marked": {{Name: "marked"}, {Name: "second"}},
	} {
		t.Run(name, func(t *testing.T) {
			provider := &fakeProvider{programs: programs}
			router, _ := makeRouter(t, provider, settings.Agent{}, EditPreview{})

			run, err := router.StartRun(t.Context(), port.Task{Question: "q"})
			if err != nil {
				t.Fatal(err)
			}
			drain(run)
			if got := provider.getTasks()[0].Agent; got != "marked" {
				t.Errorf("the default program is %q", got)
			}
		})
	}
}

// A task is refused where no program can answer it.
func TestATaskNoProgramCanAnswerIsRefused(t *testing.T) {
	for name, router := range map[string]*Router{
		"the settings name none": mustRouter(t, &fakeProvider{}, settings.Agent{Use: settings.UseProgram("")}),
		"the plugins list none":  mustRouter(t, &fakeProvider{}, settings.Agent{}),
		"there are no plugins":   mustRouter(t, nil, settings.Agent{Use: settings.UseProgram("codex")}),
	} {
		if _, err := router.StartRun(t.Context(), port.Task{Question: "q"}); err == nil {
			t.Errorf("%s: the task was taken", name)
		}
	}
}

func mustRouter(t *testing.T, provider port.AgentProvider, agent settings.Agent) *Router {
	t.Helper()
	router, _ := makeRouter(t, provider, agent, EditPreview{})
	return router
}

// The task is given the program's section of the settings, the vault's tools and where to reach
// them, and keeps what it set itself.
func TestTheTaskIsGivenWhatTheSettingsAndTheVaultHold(t *testing.T) {
	provider := &fakeProvider{}
	agent := settings.Agent{Programs: map[string]settings.AgentProgram{
		"claude": {Command: []string{"/bin/claude", "--x"}, Model: "opus", MaxSteps: 7, ShouldReadHooksAndSkills: true},
	}}
	router, _ := makeRouter(t, provider, agent, EditPreview{})

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	drain(run)
	run, err = router.StartRun(t.Context(), port.Task{
		Question: "q", Agent: "claude", Model: "haiku", MaxSteps: 3, Tools: []string{"note_read"},
	})
	if err != nil {
		t.Fatal(err)
	}
	drain(run)

	tasks := provider.getTasks()
	first, second := tasks[0], tasks[1]
	if first.Model != "opus" || first.MaxSteps != 7 || !first.ShouldReadHooksAndSkills ||
		!slices.Equal(first.Command, []string{"/bin/claude", "--x"}) {
		t.Errorf("the settings gave %+v", first)
	}
	if second.Model != "haiku" || second.MaxSteps != 3 || !slices.Equal(second.Tools, []string{"note_read"}) {
		t.Errorf("the task lost what it set: %+v", second)
	}
	if !slices.Equal(first.Tools, []string{"note_edit", "note_read"}) {
		t.Errorf("the tools are %q", first.Tools)
	}
	if first.Root != "/vault" || first.Endpoint != (port.ToolEndpoint{URL: "http://127.0.0.1:7717/mcp", Token: "secret"}) {
		t.Errorf("the vault gave %q at %+v", first.Root, first.Endpoint)
	}
	want := []port.ToolDeclaration{
		{Name: "note_edit", Title: "Edit a note", Kind: domain.StepEdit, About: "path", Match: "match", Text: "text"},
		{Name: "note_read", Title: "Read notes", Kind: domain.StepRead, About: "paths", Element: "path"},
	}
	if !slices.Equal(first.ToolDeclarations, want) {
		t.Errorf("the tools are declared as %+v, want %+v", first.ToolDeclarations, want)
	}
}

// A program the settings say nothing about is run with the step limit the defaults give.
func TestAProgramTheSettingsSayNothingAboutGetsTheDefaultStepLimit(t *testing.T) {
	provider := &fakeProvider{}
	router, _ := makeRouter(t, provider, settings.Agent{}, EditPreview{})

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "anything"})
	if err != nil {
		t.Fatal(err)
	}
	drain(run)
	if got := provider.getTasks()[0].MaxSteps; got != settings.DefaultAgentSteps {
		t.Errorf("the step limit is %d", got)
	}
}

// A plugin that cannot start the task is told to the person.
func TestWhatThePluginsRefuseIsToldWithTheTask(t *testing.T) {
	router, _ := makeRouter(t, &fakeProvider{startErr: errors.New("no such program")}, settings.Agent{}, EditPreview{})
	if _, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"}); err == nil ||
		!strings.Contains(err.Error(), "no such program") {
		t.Errorf("the error is %v", err)
	}
}

// Closing the router refuses the work that comes after it.
func TestNoProgramTakesWorkAfterTheRouterIsClosed(t *testing.T) {
	router, _ := makeRouter(t, &fakeProvider{}, settings.Agent{}, EditPreview{})
	if err := router.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"}); err == nil {
		t.Error("work was taken after the router was closed")
	}
}

// Ending a conversation is handed to the plugins, and ends nothing where there are none.
func TestEndingAConversationIsHandedToThePlugins(t *testing.T) {
	provider := &fakeProvider{}
	router, _ := makeRouter(t, provider, settings.Agent{}, EditPreview{})
	if err := router.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(provider.ended, []string{"c-1"}) {
		t.Errorf("the plugins were told %v", provider.ended)
	}

	none, _ := makeRouter(t, nil, settings.Agent{}, EditPreview{})
	if err := none.EndConversation(t.Context(), "c-1"); err != nil {
		t.Fatal(err)
	}
}

// drawn records the changes the window is told of, and the stretches looked for.
type drawn struct {
	mu      sync.Mutex
	edits   []domain.Edit
	located []string
}

func (d *drawn) preview(unique bool) EditPreview {
	return EditPreview{
		Emit: func(_ context.Context, said domain.Edit) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.edits = append(d.edits, said)
		},
		Locate: func(_ context.Context, path, stood string) (int, int, bool) {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.located = append(d.located, path+":"+stood)
			return 10, 10 + len(stood), unique
		},
	}
}

func draftOf(call, path, match, text string) port.Step {
	return port.Step{Kind: domain.StepDraft, Call: call, Place: domain.Place{Path: path}, Match: match, Text: text}
}

// A change being written is drawn where the note holds the stretch it replaces, as each draft
// arrives, and is no step the person is shown.
func TestADraftIsDrawnAndIsNoStep(t *testing.T) {
	call := port.Step{Kind: domain.StepEdit, Call: "c-1", Tool: "Edit a note"}
	provider := &fakeProvider{steps: []port.Step{
		call,
		draftOf("c-1", "a.md", "old", "ne"),
		draftOf("c-1", "a.md", "old", "new text"),
		{Kind: domain.StepStop},
	}}
	var seen drawn
	router, _ := makeRouter(t, provider, settings.Agent{}, seen.preview(true))

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	steps := drain(run)

	if len(steps) != 2 || steps[0].Kind != domain.StepEdit || steps[1].Kind != domain.StepStop {
		t.Errorf("the person is shown %+v", steps)
	}
	want := []domain.Edit{
		{Change: "c-1", Path: "a.md", From: 10, To: 13, Text: "ne"},
		{Change: "c-1", Path: "a.md", From: 10, To: 13, Text: "new text"},
	}
	if !slices.Equal(seen.edits, want) {
		t.Errorf("the window was told %+v, want %+v", seen.edits, want)
	}
	if !slices.Equal(seen.located, []string{"a.md:old"}) {
		t.Errorf("the stretch was looked for as %q, and once is enough", seen.located)
	}
}

// A stretch that stands nowhere or twice is not drawn, and is not looked for again.
func TestAStretchThatIsNotUniqueIsNotDrawn(t *testing.T) {
	provider := &fakeProvider{steps: []port.Step{
		draftOf("c-1", "a.md", "old", "x"),
		draftOf("c-1", "a.md", "old", "xy"),
	}}
	var seen drawn
	router, _ := makeRouter(t, provider, settings.Agent{}, seen.preview(false))

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if steps := drain(run); len(steps) != 0 {
		t.Errorf("the person is shown %+v", steps)
	}
	if len(seen.edits) != 0 || len(seen.located) != 1 {
		t.Errorf("drawn %+v after looking %q", seen.edits, seen.located)
	}
}

// Another call, another note or another stretch is looked for again.
func TestAnotherChangeIsLookedForAgain(t *testing.T) {
	provider := &fakeProvider{steps: []port.Step{
		draftOf("c-1", "a.md", "old", "x"),
		draftOf("c-2", "a.md", "old", "x"),
		draftOf("c-2", "b.md", "old", "x"),
		draftOf("c-2", "b.md", "older", "x"),
	}}
	var seen drawn
	router, _ := makeRouter(t, provider, settings.Agent{}, seen.preview(true))

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	drain(run)
	if want := []string{"a.md:old", "a.md:old", "b.md:old", "b.md:older"}; !slices.Equal(seen.located, want) {
		t.Errorf("looked for %q, want %q", seen.located, want)
	}
	if len(seen.edits) != 4 {
		t.Errorf("drawn %+v", seen.edits)
	}
}

// A draft naming no note or no stretch is not drawn, and neither is one where nothing can be drawn.
func TestADraftThatSaysNothingIsNotDrawn(t *testing.T) {
	steps := []port.Step{
		draftOf("c-1", "", "old", "x"),
		draftOf("c-1", "a.md", "", "x"),
		draftOf("c-1", "a.md", "old", "x"),
	}

	var seen drawn
	router, _ := makeRouter(t, &fakeProvider{steps: steps[:2]}, settings.Agent{}, seen.preview(true))
	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if shown := drain(run); len(shown) != 0 || len(seen.edits) != 0 || len(seen.located) != 0 {
		t.Errorf("shown %+v, drawn %+v, looked for %q", shown, seen.edits, seen.located)
	}

	router, _ = makeRouter(t, &fakeProvider{steps: steps[2:]}, settings.Agent{}, EditPreview{})
	run, err = router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if shown := drain(run); len(shown) != 0 {
		t.Errorf("a draft was shown as %+v", shown)
	}
}

// Stopping the run stops the plugin's, and a person who stopped reading does not hold it.
func TestStoppingTheRunStopsThePluginsRun(t *testing.T) {
	steps := make([]port.Step, 40)
	for i := range steps {
		steps[i] = port.Step{Kind: domain.StepSay, Text: "x"}
	}
	provider := &fakeProvider{steps: steps}
	router, _ := makeRouter(t, provider, settings.Agent{}, EditPreview{})

	run, err := router.StartRun(t.Context(), port.Task{Question: "q", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if err := run.Stop(); err != nil {
		t.Fatal(err)
	}
	if !provider.run.isStopped() {
		t.Error("the plugin's run goes on")
	}
	for range run.GetSteps() {
	}
	if err := run.Stop(); err != nil {
		t.Fatal(err)
	}
}

// Where no program can be chosen because a plugin could not be asked, the refusal says why.
func TestARefusalSaysWhyThePluginsCouldNotBeAsked(t *testing.T) {
	provider := &fakeProvider{listErr: errors.New("plugin md.test.one cannot be started")}
	router, _ := makeRouter(t, provider, settings.Agent{}, EditPreview{})

	_, err := router.StartRun(t.Context(), port.Task{Question: "q"})
	if err == nil || !strings.Contains(err.Error(), "cannot be started") {
		t.Errorf("the refusal is %v", err)
	}
}
