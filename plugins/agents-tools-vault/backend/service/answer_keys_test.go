package service_test

import (
	"encoding/json"
	"regexp"
	"slices"
	"testing"
)

// collectPropertyNames adds the property names of every object the JSON schema describes, at any
// depth, to names.
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

// getBranchToolNames is the tools this branch adds or changes, whose answers take camelCase keys.
func getBranchToolNames() []string {
	return []string{
		"note_list", "note_title_list", "note_resolve", "note_rename", "note_move", "note_remove",
		"link_add", "link_update", "link_remove",
		"vault_list", "vault_problem_list", "vault_add", "vault_rename", "vault_remove", "vault_open",
	}
}

func TestToolAnswersUseCamelCaseKeys(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	present := map[string]bool{}
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	for _, info := range listing.Tools {
		if !slices.Contains(getBranchToolNames(), info.GetName()) {
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
				t.Errorf("the answer of %s has the key %q, want camelCase", info.GetName(), name)
			}
		}
	}
	for _, name := range getBranchToolNames() {
		if !present[name] {
			t.Errorf("the tool %s is not listed with an output schema", name)
		}
	}
}
