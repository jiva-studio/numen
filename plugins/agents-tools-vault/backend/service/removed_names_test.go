package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

func TestSearchToolsListHasOnlyNoteSearch(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	names := map[string]bool{}
	for _, info := range infos {
		names[info.GetName()] = true
	}
	if names["search_query"] {
		t.Error("the tool search_query is listed")
	}
	if !names["note_search"] {
		t.Error("the tool note_search is not listed")
	}
}

func TestCallOfSearchQueryIsAnUnknownTool(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "search_query",
		Arguments: []byte(`{"query":"thermodynamic"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.GetError() != tools.ErrorCodeUnknownTool {
		t.Errorf("error code = %v, want unknown tool", res.GetError())
	}
}

func TestNoteSearchAnswersWithPathAndSnippetOnly(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "note_search",
		Arguments: []byte(`{"query":"thermodynamic"}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	var out struct {
		Matches []map[string]json.RawMessage `json:"matches"`
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &out); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(out.Matches) == 0 {
		t.Fatal("no matches")
	}
	for _, match := range out.Matches {
		for _, kept := range []string{"path", "snippet"} {
			if _, ok := match[kept]; !ok {
				t.Errorf("match has no %q: %v", kept, match)
			}
		}
		for _, removed := range []string{"source", "text"} {
			if _, ok := match[removed]; ok {
				t.Errorf("match has the field %q: %v", removed, match)
			}
		}
	}
}

func TestNoteWriteTakesNoBodyArgument(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	infos, err := registry.ListToolInfos(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	for _, info := range infos {
		if info.GetName() != "note_write" {
			continue
		}
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(info.GetInputSchema(), &schema); err != nil {
			t.Fatalf("unmarshal schema: %v", err)
		}
		if _, ok := schema.Properties["body"]; ok {
			t.Error("the input of note_write names body")
		}
		if _, ok := schema.Properties["content"]; !ok {
			t.Error("the input of note_write does not name content")
		}
	}
}

func TestNoteWriteWithBodyInsteadOfContentStoresNoText(t *testing.T) {
	vault, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID:   testVault,
		Name:      "note_write",
		Arguments: []byte(`{"path":"Orbit.md","body":"# Orbit\nOrbits curve."}`),
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if content, ok := vault.Content("Orbit.md"); ok && strings.Contains(content, "Orbits curve.") {
		t.Errorf("the text of the argument body was stored: %q", content)
	}
	if !res.GetResult().GetIsRefusal() {
		t.Error("a call with body and no content was not refused")
	}
}
