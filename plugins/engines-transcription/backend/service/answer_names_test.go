package service

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// decodeAnswer decodes a structured tool answer into a map of its top-level keys.
func decodeAnswer(t *testing.T, structured []byte) map[string]any {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal(structured, &answer); err != nil {
		t.Fatalf("unmarshal %s: %v", structured, err)
	}
	return answer
}

// requireKeys fails the test for every key of want the object lacks and every key of absent it has.
func requireKeys(t *testing.T, what string, object map[string]any, want, absent []string) {
	t.Helper()
	for _, key := range want {
		if _, ok := object[key]; !ok {
			t.Errorf("%s has no key %q: %v", what, key, object)
		}
	}
	for _, key := range absent {
		if _, ok := object[key]; ok {
			t.Errorf("%s has the key %q: %v", what, key, object)
		}
	}
}

// getToolSchema returns the input or the output JSON schema of the named tool.
func getToolSchema(t *testing.T, registry *tools.Registry, name string, isOutput bool) map[string]any {
	t.Helper()
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() != name {
			continue
		}
		raw := info.GetInputSchema()
		if isOutput {
			raw = info.GetOutputSchema()
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("schema of %s: %v in %s", name, err, raw)
		}
		return schema
	}
	t.Fatalf("no tool %s is registered", name)
	return nil
}

// getSchemaProperties returns the properties of the object the path of property names leads to,
// stepping into the items of an array property.
func getSchemaProperties(t *testing.T, schema map[string]any, path ...string) map[string]any {
	t.Helper()
	node := schema
	for _, name := range path {
		properties, _ := node["properties"].(map[string]any)
		next, ok := properties[name].(map[string]any)
		if !ok {
			t.Fatalf("the schema has no property %q along %v", name, path)
		}
		if items, isArray := next["items"].(map[string]any); isArray {
			next = items
		}
		node = next
	}
	properties, _ := node["properties"].(map[string]any)
	return properties
}

// visitSchema calls onProperty for every property name and onDescription for every description in
// the schema, nested objects, array items and definitions included.
func visitSchema(node any, onProperty, onDescription func(string)) {
	object, ok := node.(map[string]any)
	if !ok {
		return
	}
	if description, isText := object["description"].(string); isText {
		onDescription(description)
	}
	if properties, isObject := object["properties"].(map[string]any); isObject {
		for name, property := range properties {
			onProperty(name)
			visitSchema(property, onProperty, onDescription)
		}
	}
	for _, key := range []string{"items", "additionalProperties"} {
		visitSchema(object[key], onProperty, onDescription)
	}
	for _, key := range []string{"$defs", "definitions"} {
		if definitions, isObject := object[key].(map[string]any); isObject {
			for _, definition := range definitions {
				visitSchema(definition, onProperty, onDescription)
			}
		}
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if variants, isList := object[key].([]any); isList {
			for _, variant := range variants {
				visitSchema(variant, onProperty, onDescription)
			}
		}
	}
}

// visitEveryToolSchema runs visitSchema over the input and the output schema of every tool, and
// passes the description of each tool to onDescription as well.
func visitEveryToolSchema(t *testing.T, registry *tools.Registry, onProperty, onDescription func(tool, text string)) {
	t.Helper()
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		onDescription(info.GetName(), info.GetDescription())
		for _, raw := range [][]byte{info.GetInputSchema(), info.GetOutputSchema()} {
			if len(raw) == 0 {
				continue
			}
			var schema any
			if err := json.Unmarshal(raw, &schema); err != nil {
				t.Fatalf("schema of %s: %v in %s", info.GetName(), err, raw)
			}
			name := info.GetName()
			visitSchema(schema,
				func(text string) { onProperty(name, text) },
				func(text string) { onDescription(name, text) })
		}
	}
}

// getNarrativeMarkers returns the phrases that mark a tool or field description as narrative prose.
// Each is taken from a description these tools carried: "as the person reads it", "where in that
// file the person stands", "in front of them", "the card model it is cut by", "the passages of the
// file to light", "lights all of them", "what the tab is called", "an edit you did not see", "hand
// this to", "in words to say back to them", "the path the vault files it under", "where you speak
// about the passage".
func getNarrativeMarkers() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bperson\b`),
		regexp.MustCompile(`(?i)\bin front of\b`),
		regexp.MustCompile(`(?i)\bstand(s|ing)?\b`),
		regexp.MustCompile(`(?i)\bcut by\b`),
		regexp.MustCompile(`(?i)\blights?\b`),
		regexp.MustCompile(`(?i)\bis called\b`),
		regexp.MustCompile(`(?i)\byou did not see\b`),
		regexp.MustCompile(`(?i)\bhand this\b`),
		regexp.MustCompile(`(?i)\bsay back\b`),
		regexp.MustCompile(`(?i)\bfiles it under\b`),
		regexp.MustCompile(`(?i)\bspeak\b`),
	}
}

func TestTranscriptCreateReturnsTheTaskIDKey(t *testing.T) {
	s := newToolScene(t)
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Lectures", Name: "transcript_create", Arguments: []byte(`{"path":"` + lecture + `"}`),
	})
	if res.GetResult().GetIsRefusal() {
		t.Fatalf("transcript_create was refused: %s", res.GetResult().GetText())
	}
	answer := decodeAnswer(t, res.GetResult().GetStructured())
	requireKeys(t, "the answer of transcript_create", answer, []string{"taskId", "path", "state"}, []string{"task_id"})
	if want := "transcription:" + lecture; answer["taskId"] != want {
		t.Errorf("taskId = %v, want %s", answer["taskId"], want)
	}
	close(s.blocking.release)
	s.waitFinished(t, "transcription:"+lecture)

	schema := getToolSchema(t, s.registry, "transcript_create", true)
	requireKeys(t, "the output schema of transcript_create", getSchemaProperties(t, schema), []string{"taskId"}, []string{"task_id"})
}

func TestTranscriptionToolSchemasNameEveryKeyInCamelCase(t *testing.T) {
	s := newToolScene(t)
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if strings.Contains(name, "_") {
			t.Errorf("the schema of %s names the key %q, which is not camelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestTranscriptionToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	s := newToolScene(t)
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, s.registry, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}
