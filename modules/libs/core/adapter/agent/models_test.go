package agent_test

import (
	"os"
	"path/filepath"
	"slices"
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

// getNames is the name of each model, in the order the panel draws them.
func getNames(models []port.Model) []string {
	names := make([]string, 0, len(models))
	for _, m := range models {
		names = append(names, m.Name)
	}
	return names
}

// checkWrites holds that choosing a model writes its own name at the path the
// program's model setting stands at.
func checkWrites(t *testing.T, models []port.Model, at []string) {
	t.Helper()
	for _, m := range models {
		want := `"` + m.Name + `"`
		if len(m.Writes) != 1 || !slices.Equal(m.Writes[0].Path, at) || m.Writes[0].JSON != want {
			t.Errorf("model %q writes %v, want %s at %v", m.Name, m.Writes, want, at)
		}
		if !slices.Equal(m.Path, at) {
			t.Errorf("model %q stands at %v, want %v", m.Name, m.Path, at)
		}
	}
}

// A model the settings name that the list does not hold is added after the
// listed ones; one it already holds, or none, adds nothing.
func TestAConfiguredModelIsListedOnceAfterTheOnesTheAgentOffers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	for _, c := range []struct {
		use   string
		model func(*agent.Config, string)
		at    []string
		known []string
	}{
		{agent.UseClaude, func(h *agent.Config, m string) { h.Claude.Model = m }, []string{"claude", "model"},
			[]string{"", "opus", "sonnet", "haiku", "claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"}},
		{agent.UseAntigravity, func(h *agent.Config, m string) { h.Antigravity.Model = m }, []string{"antigravity", "model"},
			[]string{"gemini-3.7-flash", "gemini-2.5-pro", "gemini-2.5-flash"}},
		{agent.UseCodex, func(h *agent.Config, m string) { h.Codex.Model = m }, []string{"codex", "model"},
			[]string{"gpt-5.6-terra", "gpt-5.6-luna"}},
	} {
		t.Run(c.use, func(t *testing.T) {
			for _, named := range []string{"", c.known[len(c.known)-1]} {
				held := agent.Defaults()
				held.Use = c.use
				c.model(&held, named)
				models := agent.GetModels(held)
				if got := getNames(models); !slices.Equal(got, c.known) {
					t.Errorf("model %q named: listed %q, want %q", named, got, c.known)
				}
				checkWrites(t, models, c.at)
			}

			held := agent.Defaults()
			held.Use = c.use
			c.model(&held, "house-model")
			models := agent.GetModels(held)
			if got, want := getNames(models), append(slices.Clone(c.known), "house-model"); !slices.Equal(got, want) {
				t.Fatalf("listed %q, want %q", got, want)
			}
			added := models[len(models)-1]
			if added.Title != "house-model" || added.IsDefault || added.Presence != port.NothingToFetch {
				t.Errorf("the model the settings name is listed as %+v", added)
			}
			if added.Shelf == "" || added.Shelf == models[len(models)-2].Shelf {
				t.Errorf("the model the settings name stands on shelf %q, beside %q", added.Shelf, models[len(models)-2].Shelf)
			}
			checkWrites(t, models, c.at)
		})
	}
}

func TestTheClaudeModelsStandOnTheirShelves(t *testing.T) {
	held := agent.Defaults()
	held.Use = agent.UseClaude
	models := agent.GetModels(held)
	if !models[0].IsDefault || models[0].Name != "" {
		t.Errorf("the first model is %+v, want the default", models[0])
	}
	shelves := map[string][]string{}
	for _, m := range models[1:] {
		shelves[m.Shelf] = append(shelves[m.Shelf], m.Name)
		if m.Title != m.Name || m.IsDefault || m.Shelf == "" {
			t.Errorf("model %q is listed as %+v", m.Name, m)
		}
	}
	if len(shelves) != 2 {
		t.Fatalf("models stand on %d shelves, want 2: %v", len(shelves), shelves)
	}
	for _, names := range shelves {
		switch names[0] {
		case "opus":
			if !slices.Equal(names, []string{"opus", "sonnet", "haiku"}) {
				t.Errorf("shelf of sizes holds %q", names)
			}
		case "claude-opus-5":
			if !slices.Equal(names, []string{"claude-opus-5", "claude-sonnet-5", "claude-haiku-4-5"}) {
				t.Errorf("shelf of full names holds %q", names)
			}
		default:
			t.Errorf("unexpected shelf %q", names)
		}
	}
}

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
