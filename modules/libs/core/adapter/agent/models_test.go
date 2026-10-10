package agent_test

import (
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

func TestGetModelsReturnsPresence(t *testing.T) {
	for _, use := range []string{agent.UseClaude, agent.UseAntigravity, agent.UseCodex} {
		held := agent.Defaults()
		held.Use = use
		models := agent.GetModels(held)
		if len(models) == 0 {
			t.Fatalf("expected models for %s, got 0", use)
		}

		for _, m := range models {
			if m.Presence != port.NothingToFetch {
				t.Errorf("model %s (%s) presence = %v, expected %v", m.Name, use, m.Presence, port.NothingToFetch)
			}
		}
	}
}
