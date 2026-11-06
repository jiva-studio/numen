package service_test

import (
	"encoding/json"
	"regexp"
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

func TestToolAnswersUseCamelCaseKeys(t *testing.T) {
	_, _, registry := newTwoVaults(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	camelCase := regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	for _, info := range listing.Tools {
		raw := info.GetOutputSchema()
		if len(raw) == 0 {
			continue
		}
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
}
