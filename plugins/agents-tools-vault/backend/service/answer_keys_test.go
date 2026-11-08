package service_test

import (
	"encoding/json"
	"regexp"
	"slices"
	"testing"
)

func collectPropertyNames(schema any, names map[string]bool) {
	switch node := schema.(type) {
	case map[string]any:
		if properties, ok := node["properties"].(map[string]any); ok {
			for name, sub := range properties {
				names[name] = true
				collectPropertyNames(sub, names)
			}
		}
		for key, sub := range node {
			if key != "properties" {
				collectPropertyNames(sub, names)
			}
		}
	case []any:
		for _, sub := range node {
			collectPropertyNames(sub, names)
		}
	}
}

func getCamelCaseToolNames() []string {
	return []string{
		"note_list", "note_title_list", "note_resolve", "note_rename", "note_move", "note_remove",
		"link_add", "link_update", "link_remove",
		"vault_list", "vault_problem_list", "vault_add", "vault_rename", "vault_remove", "vault_open",
	}
}

func TestToolResultsUseCamelCaseKeys(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	present := map[string]bool{}
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	for _, info := range listing.Tools {
		if !slices.Contains(getCamelCaseToolNames(), info.GetName()) {
			continue
		}
		raw := info.GetOutputSchema()
		if len(raw) == 0 {
			continue
		}
		present[info.GetName()] = true
		var schema any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("the output schema of %s: %v", info.GetName(), err)
		}
		names := map[string]bool{}
		collectPropertyNames(schema, names)
		for name := range names {
			if !camelCase.MatchString(name) {
				t.Errorf("the result of %s has the key %q, want camelCase", info.GetName(), name)
			}
		}
	}
	for _, name := range getCamelCaseToolNames() {
		if !present[name] {
			t.Errorf("the tool %s is not listed with an output schema", name)
		}
	}
}
