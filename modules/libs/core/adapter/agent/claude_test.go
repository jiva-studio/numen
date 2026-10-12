package agent_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
)

func TestTheClaudeModelsStandOnTheirShelves(t *testing.T) {
	held := agent.Defaults()
	held.Use = agent.UseClaude
	models := agent.GetModels(held)
	if !models[0].IsDefault || models[0].Name != "" {
		t.Errorf("the first model is %+v, want the default", models[0])
	}
	groups := map[string][]string{}
	for _, m := range models[1:] {
		groups[m.Group] = append(groups[m.Group], m.Name)
		if m.Title != m.Name || m.IsDefault || m.Group == "" {
			t.Errorf("model %q is listed as %+v", m.Name, m)
		}
	}
	if len(groups) != 2 {
		t.Fatalf("models stand on %d groups, want 2: %v", len(groups), groups)
	}
	for _, names := range groups {
		switch names[0] {
		case "opus":
			if !slices.Equal(names, []string{"opus", "sonnet", "haiku"}) {
				t.Errorf("group of sizes holds %q", names)
			}
		case "claude-opus-5":
			if !slices.Equal(names, []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"}) {
				t.Errorf("group of full names holds %q", names)
			}
		default:
			t.Errorf("unexpected group %q", names)
		}
	}
}
