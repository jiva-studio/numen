package service

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func decodeAnswer(t *testing.T, structured []byte) map[string]any {
	t.Helper()
	var answer map[string]any
	if err := json.Unmarshal(structured, &answer); err != nil {
		t.Fatalf("unmarshal %s: %v", structured, err)
	}
	return answer
}

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

func getSchemaProperties(t *testing.T, schema map[string]any, path ...string) map[string]any {
	t.Helper()
	node := schema
	for _, name := range path {
		properties, hasProperties := node["properties"].(map[string]any)
		if !hasProperties {
			t.Fatalf("the schema has no properties before %q along %v", name, path)
		}
		next, ok := properties[name].(map[string]any)
		if !ok {
			t.Fatalf("the schema has no property %q along %v", name, path)
		}
		if items, isArray := next["items"].(map[string]any); isArray {
			next = items
		}
		node = next
	}
	properties, ok := node["properties"].(map[string]any)
	if !ok {
		t.Fatalf("the schema has no properties at the end of %v", path)
	}
	return properties
}

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
		regexp.MustCompile(`(?i)\bhold(s|ing)?\b`),
		regexp.MustCompile(`(?i)\breach(es|ing)?\b`),
		regexp.MustCompile(`(?i)\bcarr(y|ies|ied|ying)\b`),
		regexp.MustCompile(`(?i)\brefus(e|es|ed|ing)\b`),
		regexp.MustCompile(`(?i)\blies inside\b`),
		regexp.MustCompile(`(?i)\bgoes to\b`),
		regexp.MustCompile(`(?i)\blit\b`),
		regexp.MustCompile(`(?i)\btak(e|es|ing)\b[^.;]*\boff\b`),
	}
}

func TestRecognitionCreateReturnsTheTaskIDKey(t *testing.T) {
	s := newToolFixture(t)
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Physics", Name: "recognition_create", Arguments: []byte(`{"path":"` + scanPath + `"}`),
	})
	if res.GetResult().GetIsError() {
		t.Fatalf("recognition_create was rejected: %s", res.GetResult().GetText())
	}
	answer := decodeAnswer(t, res.GetResult().GetStructured())
	requireKeys(t, "the answer of recognition_create", answer, []string{"taskId", "path", "state"}, []string{"task_id"})
	if want := "recognize:Physics:" + scanPath; answer["taskId"] != want {
		t.Errorf("taskId = %v, want %s", answer["taskId"], want)
	}
	close(s.recognizer.release)
	s.waitUntilFinished(t, "recognize:Physics:"+scanPath)

	schema := getToolSchema(t, s.registry, "recognition_create", true)
	requireKeys(t, "the output schema of recognition_create", getSchemaProperties(t, schema), []string{"taskId"}, []string{"task_id"})
}

func TestOCRToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	s := newToolFixture(t)
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, s.registry, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}

func TestOCRToolInstructionsContainNoNarrativeMarker(t *testing.T) {
	s := newToolFixture(t)
	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(getToolSentences(listing, "recognition_create")) == 0 {
		t.Fatal("recognition_create lists no instructions")
	}
	for _, info := range listing.Tools {
		for _, sentence := range info.Instructions {
			for _, marker := range getNarrativeMarkers() {
				if marker.MatchString(sentence) {
					t.Errorf("an instruction of %s matches %s: %q", info.GetName(), marker, sentence)
				}
			}
		}
	}
}

func getToolSentences(listing tools.Listing, name string) []string {
	for _, info := range listing.Tools {
		if info.GetName() == name {
			return info.Instructions
		}
	}
	return nil
}

func isOneSentence(text string) bool {
	return strings.TrimSpace(text) == text && strings.HasSuffix(text, ".") && !regexp.MustCompile(`[.!?]\s+\S`).MatchString(text)
}

// lowerCamelCase is the form of every property name of a tool schema.
var lowerCamelCase = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestOCRToolSchemasNameEveryPropertyInLowerCamelCase(t *testing.T) {
	s := newToolFixture(t)
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if !lowerCamelCase.MatchString(name) {
			t.Errorf("the schema of %s names the property %q, which is not lowerCamelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestOCRToolTextsNameNoToolOfAnotherPlugin(t *testing.T) {
	s := newToolFixture(t)
	foreign := getForeignToolNames()
	visitEveryToolSchema(t, s.registry, func(string, string) {}, func(tool, text string) {
		for _, name := range foreign {
			if strings.Contains(text, name) {
				t.Errorf("a description of %s or its schema names %s: %q", tool, name, text)
			}
		}
	})
	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		for _, sentence := range info.Instructions {
			for _, name := range foreign {
				if strings.Contains(sentence, name) {
					t.Errorf("an instruction of %s names %s: %q", info.GetName(), name, sentence)
				}
			}
		}
	}
}

func getForeignToolNames() []string {
	all := []string{
		"note_read", "note_write", "note_heading_list", "note_search", "note_list", "note_title_list",
		"note_resolve", "note_rename", "note_move", "note_remove", "link_list", "link_add", "link_update",
		"link_remove", "file_read", "file_create", "library_read", "library_search", "vault_get", "vault_list",
		"vault_problem_list", "vault_add", "vault_rename", "vault_remove", "vault_open",
		"note_focus", "source_focus", "window_tab_list", "window_tab_activate", "note_edit_preview",
		"recognition_create", "transcript_create", "url_import",
		"card_add", "card_read", "card_edit", "card_remove", "card_current_get", "card_deck_create",
	}
	return slices.DeleteFunc(all, func(name string) bool { return name == "recognition_create" })
}
