package agent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestGetProgramsReturnsPresence(t *testing.T) {
	programs := agent.GetPrograms()
	if len(programs) != 4 {
		t.Fatalf("expected 4 programs, got %d", len(programs))
	}

	for _, p := range programs {
		switch p.Name {
		case agent.UseClaude:
			expected := port.NotFetched
			if agent.IsClaudeInstalled() {
				expected = port.Present
			}
			if p.Presence != expected {
				t.Errorf("claude presence = %v, expected %v", p.Presence, expected)
			}
		case agent.UseAntigravity:
			expected := port.NotFetched
			if agent.IsAntigravityInstalled() {
				expected = port.Present
			}
			if p.Presence != expected {
				t.Errorf("antigravity presence = %v, expected %v", p.Presence, expected)
			}
		case agent.UseCodex:
			expected := port.NotFetched
			if agent.IsCodexInstalled() {
				expected = port.Present
			}
			if p.Presence != expected {
				t.Errorf("codex presence = %v, expected %v", p.Presence, expected)
			}
		case "":
			if p.Presence != port.NothingToFetch {
				t.Errorf("nothing answers presence = %v, expected %v", p.Presence, port.NothingToFetch)
			}
		}
	}
}

func TestAProgramOnThePathIsPresent(t *testing.T) {
	for _, c := range []struct {
		command string
		use     string
	}{
		{"claude", agent.UseClaude},
		{"agy", agent.UseAntigravity},
		{"codex", agent.UseCodex},
	} {
		t.Run(c.use, func(t *testing.T) {
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, c.command), []byte("#!/bin/sh\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin)
			for _, p := range agent.GetPrograms() {
				if p.Name == c.use && p.Presence != port.Present {
					t.Errorf("%s is on the path and its presence is %v", c.use, p.Presence)
				}
			}
		})
	}
}

func TestGetModelsProvidesFallbackForKnownAgents(t *testing.T) {
	cfg := agent.Defaults()
	models := agent.GetModels(cfg)
	if len(models) == 0 {
		t.Fatalf("expected models to be returned, got empty slice")
	}

	foundClaudeModel := false
	for _, m := range models {
		if len(m.Path) == 2 && m.Path[0] == "claude" && m.Path[1] == "model" {
			foundClaudeModel = true
			break
		}
	}
	if !foundClaudeModel {
		t.Errorf("expected at least one claude model in catalogue, got none")
	}
}
