package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun"
)

func TestResolveCommandWithEmptyCommand(t *testing.T) {
	withoutAgentsAnywhere(t)

	// An empty command falls through to FindCodex, FindCursor, or FindAntigravity.
	// Mutant len(params.Command) > -1 panics when params.Command is empty.
	bin, args := resolveCommand(RunParams{Program: "codex"})
	if bin != "codex" || len(args) != 0 {
		t.Errorf("expected codex and empty args, got %s, %v", bin, args)
	}

	bin, args = resolveCommand(RunParams{Program: "cursor"})
	if bin != "cursor" || len(args) != 0 {
		t.Errorf("expected cursor and empty args, got %s, %v", bin, args)
	}

	bin, args = resolveCommand(RunParams{Program: "antigravity"})
	if bin != "agy" || len(args) != 0 {
		t.Errorf("expected agy and empty args, got %s, %v", bin, args)
	}
}

func TestStartRunWorkBufferCapacity(t *testing.T) {
	needsShell(t)
	script := writeScript(t, "runner", "exit 0\n")
	runner := &AgentRunner{Registry: &agentrun.Registry{}}

	w, err := runner.StartRun(t.Context(), RunParams{
		Program: "custom",
		Command: []string{script},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	if got := cap(w.Output); got != 64 {
		t.Errorf("expected work output channel capacity 64, got %d", got)
	}
}

func TestConfigureCommandTokenEnv(t *testing.T) {
	needsShell(t)
	out := filepath.Join(t.TempDir(), "token")
	script := writeScript(t, "runner", "printf '%s' \"$NUMEN_TOKEN\" > "+quoteShell(out)+"\n")
	runner := &AgentRunner{Registry: &agentrun.Registry{}}

	w, err := runner.StartRun(t.Context(), RunParams{
		Program: "custom",
		Command: []string{script},
		Tools:   Endpoint{Token: "test-token-xyz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Stop() }()

	// Wait for the script to finish and write out.
	for range w.Output {
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "test-token-xyz" {
		t.Errorf("expected NUMEN_TOKEN 'test-token-xyz', got %q", string(data))
	}
}

func TestStartRunProgramDetection(t *testing.T) {
	needsShell(t)

	cases := []struct {
		name         string
		program      string
		scriptName   string
		wantCodex    bool
		wantAntigrav bool
	}{
		{
			name:         "antigravity program name with arbitrary binary",
			program:      "antigravity",
			scriptName:   "custom-runner",
			wantAntigrav: true,
		},
		{
			name:         "arbitrary program with agy suffix binary",
			program:      "custom",
			scriptName:   "bin-agy",
			wantAntigrav: true,
		},
		{
			name:         "arbitrary program with antigravity suffix binary",
			program:      "custom",
			scriptName:   "bin-antigravity",
			wantAntigrav: true,
		},
		{
			name:       "codex program name with arbitrary binary",
			program:    "codex",
			scriptName: "custom-runner",
			wantCodex:  true,
		},
		{
			name:       "arbitrary program with codex suffix binary",
			program:    "custom",
			scriptName: "bin-codex",
			wantCodex:  true,
		},
		{
			name:       "arbitrary program with arbitrary binary",
			program:    "custom",
			scriptName: "custom-runner",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "args")
			script := writeScript(t, tc.scriptName, "for a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+quoteShell(out)+"\n")
			runner := &AgentRunner{Registry: &agentrun.Registry{}}

			w, err := runner.StartRun(t.Context(), RunParams{
				Program: tc.program,
				Command: []string{script},
				Prompt:  "test-prompt",
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = w.Stop() }()

			for range w.Output {
			}

			data, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")

			isCodex := slices.Contains(lines, "exec")
			isAntigrav := slices.Contains(lines, "--output-format")

			if isCodex != tc.wantCodex {
				t.Errorf("got isCodex=%v, want %v (lines: %v)", isCodex, tc.wantCodex, lines)
			}
			if isAntigrav != tc.wantAntigrav {
				t.Errorf("got isAntigrav=%v, want %v (lines: %v)", isAntigrav, tc.wantAntigrav, lines)
			}
		})
	}
}
