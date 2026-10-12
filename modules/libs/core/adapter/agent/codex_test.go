package agent_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
)

// The models the Codex cache lists are offered, hidden ones and ones with no
// slug left out; the first is the default.
func TestTheCodexModelsAreReadFromItsCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	cache := `{"models":[
		{"slug":"alpha","display_name":"Alpha One","visibility":"list"},
		{"slug":"secret","display_name":"Secret","visibility":"hide"},
		{"slug":"","display_name":"Nameless","visibility":"list"},
		{"slug":"beta","display_name":"","visibility":"list"}
	]}`
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(cache), 0o644); err != nil {
		t.Fatal(err)
	}

	held := agent.Defaults()
	held.Use = agent.UseCodex
	held.Codex.Model = "house-model"
	models := agent.GetModels(held)
	if got, want := getNames(models), []string{"alpha", "beta", "house-model"}; !slices.Equal(got, want) {
		t.Fatalf("listed %q, want %q", got, want)
	}
	if !models[0].IsDefault || models[0].Title != "Default (Alpha One)" {
		t.Errorf("the first model is %+v", models[0])
	}
	if models[1].IsDefault || models[1].Title != "beta" {
		t.Errorf("a model with no display name is %+v", models[1])
	}
	checkWrites(t, models, []string{"codex", "model"})
}
