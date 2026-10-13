package acp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/acp"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestAgent_FinishEmpty(t *testing.T) {
	agent := &acp.Agent{Program: "antigravity"}
	if err := agent.EndConversation(t.Context(), ""); err != nil {
		t.Errorf("unexpected error on empty finish: %v", err)
	}
}

func TestAgent_TakeWithoutTools(t *testing.T) {
	agent := &acp.Agent{Program: "antigravity"}
	_, err := agent.StartRun(t.Context(), port.Task{Question: "hello"})
	if err == nil {
		t.Errorf("expected error when no tools provided")
	}
}

// makeCodex is a codex agent whose program writes its arguments down, prints five finished tool
// calls and then waits to be killed.
func makeCodex(t *testing.T) (agent *acp.Agent, argv string) {
	t.Helper()

	dir := t.TempDir()
	argv = filepath.Join(dir, "argv")
	script := filepath.Join(dir, "codex")
	body := "#!/bin/sh\n" +
		"for a in \"$@\"; do printf '%s\\n' \"$a\"; done > " + argv + "\n" +
		"for i in 1 2 3 4 5; do\n" +
		"  echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"mcp_tool_call\"}}'\n" +
		"done\n" +
		"sleep 60\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	agent = acp.New(acp.Options{
		Program: "codex",
		Root:    dir,
		Tools:   acp.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	})
	agent.Command = []string{script}
	t.Cleanup(func() { _ = agent.Close() })
	return agent, argv
}

// readSteps drains a run, failing when it does not end.
func readSteps(t *testing.T, run port.Run) []port.Step {
	t.Helper()

	var steps []port.Step
	deadline := time.After(10 * time.Second)
	for {
		select {
		case step, ok := <-run.GetSteps():
			if !ok {
				return steps
			}
			steps = append(steps, step)
		case <-deadline:
			t.Fatalf("the run did not end: %+v", steps)
		}
	}
}

// The task's step limit stops the agent after that many finished tool calls.
func TestMaxStepsStopsTheAgent(t *testing.T) {
	agent, _ := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q", MaxSteps: 2})
	if err != nil {
		t.Fatal(err)
	}
	steps := readSteps(t, run)

	var done int
	for _, step := range steps {
		if step.Kind == domain.StepToolDone {
			done++
		}
	}
	if done != 2 {
		t.Errorf("%d tool calls finished, want 2", done)
	}
	last := steps[len(steps)-1]
	if last.Kind != domain.StepStop || !strings.Contains(last.Detail, "2 steps") {
		t.Errorf("last step is %+v, want a stop naming the limit", last)
	}
}

// A task naming no limit runs to the end of what the agent prints.
func TestNoMaxStepsLetsTheAgentFinishItsOutput(t *testing.T) {
	agent, _ := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	var done int
	for step := range run.GetSteps() {
		if step.Kind == domain.StepToolDone {
			done++
		}
		if done == 5 {
			break
		}
	}
	if done != 5 {
		t.Errorf("%d tool calls finished, want 5", done)
	}
	_ = run.Stop()
}

// The instructions of the task go to the agent ahead of the question.
func TestInstructionsAreGivenAheadOfTheQuestion(t *testing.T) {
	agent, argv := makeCodex(t)

	run, err := agent.StartRun(t.Context(), port.Task{Question: "what is here?", Instructions: "Call a note by its title.", MaxSteps: 1})
	if err != nil {
		t.Fatal(err)
	}
	readSteps(t, run)

	raw, err := os.ReadFile(argv)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if got, want := strings.Join(lines[len(lines)-2:], "\n"), "Call a note by its title.\nwhat is here?"; got != want {
		t.Errorf("the agent was asked %q, want %q", got, want)
	}
}

// Stopping a run ends it although nobody reads its steps and its buffer is full.
func TestStopEndsARunWhoseStepsAreNotRead(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "codex")
	body := "#!/bin/sh\n" +
		"for i in $(seq 1 200); do\n" +
		"  echo '{\"type\":\"item.completed\",\"item\":{\"type\":\"mcp_tool_call\"}}'\n" +
		"done\n" +
		"sleep 60\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	agent := acp.New(acp.Options{
		Program: "codex",
		Root:    dir,
		Tools:   acp.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
	})
	agent.Command = []string{script}
	t.Cleanup(func() { _ = agent.Close() })

	run, err := agent.StartRun(t.Context(), port.Task{Question: "q"})
	if err != nil {
		t.Fatal(err)
	}
	for len(run.GetSteps()) < cap(run.GetSteps()) {
		time.Sleep(10 * time.Millisecond)
	}

	stopped := make(chan struct{})
	go func() {
		_ = run.Stop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return")
	}
}
