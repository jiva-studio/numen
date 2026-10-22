package container_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"

	"github.com/jiva-studio/numen/modules/libs/core/port/completionport"
)

// serveCompletions gives the host the settings written and returns what answers the questions of
// its plugins.
func serveCompletions(t *testing.T, host *plugin.Host, output string) (completionport.Completer, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := container.Config{PluginHost: host, SettingsPath: path}
	target, err := cfg.Settings()
	if err != nil {
		t.Fatal(err)
	}
	cfg.SetSettings(target)
	completer := host.GetCompleter()
	if completer == nil {
		t.Fatal("the host has no completer")
	}
	return completer, path
}

func ask(text string) completionport.Question {
	return completionport.Question{Messages: []completionport.Message{{Role: "system", Text: "be brief"}, {Role: "user", Text: text}}}
}

// A question goes to the program the default names as a task with no tool and no conversation, under
// the program's section of the settings, and the answer is what the program said.
func TestAQuestionIsATaskOfTheProgramTheDefaultNames(t *testing.T) {
	host, one, _ := twoProviders(t)
	completer, _ := serveCompletions(t, host, `{
		"agent": {"alpha": {"command": ["/bin/alpha", "--x"], "model": "section-model"}},
		"ai": {"default": {"agent": "alpha"}}
	}`)

	got, err := completer.Complete(t.Context(), "md.test.asking", ask("why?"))
	if err != nil || got != "answer: why?" {
		t.Fatalf("answered %q, %v", got, err)
	}

	request := one.getAsked()
	if len(request) != 1 {
		t.Fatalf("%d tasks, want 1", len(request))
	}
	task := request[0]
	if task.GetAgent() != "alpha" || task.GetModel() != "section-model" || task.GetInstructions() != "be brief" ||
		!slices.Equal(task.GetCommand(), []string{"/bin/alpha", "--x"}) {
		t.Errorf("the task is %v", task)
	}
	if len(task.GetTools()) != 0 || len(task.GetToolDeclarations()) != 0 || task.GetConversation() != "" ||
		task.GetToolsUrl() != "" || task.GetToolsToken() != "" {
		t.Errorf("the task holds tools or a conversation: %v", task)
	}
}

// is the program the plugins mark as the default.
func TestTheRouteModelExistsOverTheSectionAndAnEmptyDefaultIsTheProgramsDefault(t *testing.T) {
	host, one, _ := twoProviders(t)
	completer, _ := serveCompletions(t, host, `{
		"agent": {"alpha": {"model": "section-model"}},
		"ai": {"default": {"model": "big"}}
	}`)

	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); err != nil {
		t.Fatal(err)
	}

	request := one.getAsked()
	if len(request) != 1 || request[0].GetAgent() != "alpha" || request[0].GetModel() != "big" {
		t.Errorf("the tasks are %v", request)
	}
}

// An empty default is the program agent.use names, and none when the settings name none.
func TestAnEmptyDefaultFollowsTheAgentTheSettingsUse(t *testing.T) {
	host, one, _ := twoProviders(t)
	named, _ := serveCompletions(t, host, `{"agent": {"use": "alpha"}}`)
	if _, err := named.Complete(t.Context(), "md.test.asking", ask("why?")); err != nil {
		t.Fatal(err)
	}
	if request := one.getAsked(); len(request) != 1 || request[0].GetAgent() != "alpha" {
		t.Errorf("the tasks are %v", request)
	}

	none, _ := serveCompletions(t, host, `{"agent": {"use": ""}}`)
	if _, err := none.Complete(t.Context(), "md.test.asking", ask("why?")); !errors.Is(err, completionport.ErrNoRoute) {
		t.Errorf("with no agent named, error = %v, want port.ErrNoRoute", err)
	}
}

// A program no plugin lists, one that is not installed, and a host with no plugin that runs
// programs are all no route, and no other program answers in their place.
func TestAProgramThatIsNotThereIsNoRoute(t *testing.T) {
	host, one, two := twoProviders(t)
	for _, content := range []string{
		`{"ai": {"default": {"agent": "ghost"}}}`,
		`{"ai": {"default": {"agent": "beta"}}}`,
	} {
		completer, _ := serveCompletions(t, host, content)
		if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); !errors.Is(err, completionport.ErrNoRoute) {
			t.Errorf("%s: error = %v, want port.ErrNoRoute", content, err)
		}
	}
	if len(one.getAsked()) != 0 || len(two.getAsked()) != 0 {
		t.Errorf("a task was started: %v, %v", one.getAsked(), two.getAsked())
	}

	empty := hostWith(t, map[string][]string{"md.test.chatty": {"agent"}}, nil)
	completer, _ := serveCompletions(t, empty, `{}`)
	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); !errors.Is(err, completionport.ErrNoRoute) {
		t.Errorf("without a provider, error = %v, want port.ErrNoRoute", err)
	}
}

// A question routed to a program that no plugin lists, while a plugin that runs programs cannot be
// started, is unavailable.
func TestAProviderThatCannotBeStartedIsUnavailable(t *testing.T) {
	host := hostWith(t, map[string][]string{providerOne: {"ai.agent.provider"}}, nil)
	completer, _ := serveCompletions(t, host, `{"ai": {"default": {"agent": "alpha"}}}`)

	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); !errors.Is(err, completionport.ErrUnavailable) {
		t.Errorf("error = %v, want port.ErrUnavailable", err)
	}
}

// The default is read from the settings file at every question, so a default written while the
// application runs applies to the next one.
func TestADefaultSavedToTheSettingsAppliesToTheNextQuestion(t *testing.T) {
	host, one, _ := twoProviders(t)
	completer, path := serveCompletions(t, host, `{"ai": {"default": {"agent": "ghost"}}}`)
	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); !errors.Is(err, completionport.ErrNoRoute) {
		t.Fatalf("with a default that is not there: error = %v, want port.ErrNoRoute", err)
	}

	if err := os.WriteFile(path, []byte(`{"ai": {"default": {"agent": "alpha", "model": "big"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); err != nil {
		t.Fatal(err)
	}
	if request := one.getAsked(); len(request) != 1 || request[0].GetModel() != "big" {
		t.Errorf("the tasks are %v", request)
	}
}

// A model the program no longer lists is replaced by the one it answers with unasked.
func TestAModelTheProgramNoLongerListsIsReplacedByItsDefault(t *testing.T) {
	host, one, _ := twoProviders(t)
	completer, _ := serveCompletions(t, host, `{"ai": {"default": {"agent": "alpha", "model": "retired"}}}`)

	if _, err := completer.Complete(t.Context(), "md.test.asking", ask("why?")); err != nil {
		t.Fatal(err)
	}

	if request := one.getAsked(); len(request) != 1 || request[0].GetModel() != "big" {
		t.Errorf("the tasks are %v", request)
	}
}
