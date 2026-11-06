package cardtools_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

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

// requireList returns the value as a JSON array and fails the test when it is not one.
func requireList(t *testing.T, what string, value any) []any {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is %v, not an array", what, value)
	}
	return list
}

// requireObject returns the value as a JSON object and fails the test when it is not one.
func requireObject(t *testing.T, what string, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is %v, not an object", what, value)
	}
	return object
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

func TestCardReadReturnsTheProblemsKeyWithAMessage(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_read", map[string]any{"path": "Animals.md"})
	requireKeys(t, "the answer of card_read", answer, []string{"problems"}, []string{"faults"})
	problems := requireList(t, "answer[problems]", answer["problems"])
	if len(problems) != 1 {
		t.Fatalf("problems = %v, want one", answer["problems"])
	}
	problem := requireObject(t, "problems[0]", problems[0])
	requireKeys(t, "a problem of card_read", problem, []string{"cardIndex", "message"}, []string{"why", "card"})
	if message, isText := problem["message"].(string); !isText || message == "" {
		t.Errorf("the problem %v has an empty message", problem)
	}
	// The second card of the fixture deck, Alpaca, has no card model link.
	if index, isNumber := problem["cardIndex"].(float64); !isNumber || index != 1 {
		t.Errorf("cardIndex = %v, want 1", problem["cardIndex"])
	}

	schema := getToolSchema(t, s.registry, "card_read", true)
	requireKeys(t, "the output schema of card_read", getSchemaProperties(t, schema), []string{"problems"}, []string{"faults"})
	requireKeys(t, "the output schema of a card_read problem", getSchemaProperties(t, schema, "problems"), []string{"cardIndex", "message"}, []string{"why", "card"})
}

func TestCardFieldRenameAnswersTheUpdatedDecksAndHeadings(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_field_rename", map[string]any{
		"path": "Animal.md", "from": "Height", "to": "Shoulder height",
	})
	requireKeys(t, "the answer of card_field_rename", answer, []string{"updatedDecks", "updatedHeadings"}, []string{"decks", "cards", "notWritten"})
	if decks := requireList(t, "answer[updatedDecks]", answer["updatedDecks"]); len(decks) != 1 || decks[0] != "Animals.md" {
		t.Errorf("updatedDecks = %v, want [Animals.md]", answer["updatedDecks"])
	}
	if answer["updatedHeadings"] != float64(1) {
		t.Errorf("updatedHeadings = %v, want 1", answer["updatedHeadings"])
	}

	schema := getToolSchema(t, s.registry, "card_field_rename", true)
	requireKeys(t, "the output schema of card_field_rename", getSchemaProperties(t, schema),
		[]string{"updatedDecks", "updatedHeadings", "failedDecks"}, []string{"decks", "cards", "notWritten"})
}

func TestCardFieldRenameReturnsTheFailedDecksKey(t *testing.T) {
	notes := vault()
	notes["Herds.md"] = "---\ntype: deck\n---\n\n## Guanaco ^zpqrstvwxy\n\n[[Animal]]\n\n### Height\n\nabout 44\"\n"
	s, v := newSession(t, notes)
	if err := os.Remove(filepath.Join(v.Path, "Herds.md")); err != nil {
		t.Fatal(err)
	}
	answer := call[map[string]any](t, s, "card_field_rename", map[string]any{
		"path": "Animal.md", "from": "Height", "to": "Stature",
	})
	requireKeys(t, "the answer of card_field_rename", answer, []string{"failedDecks"}, []string{"notWritten"})
	if failed := requireList(t, "answer[failedDecks]", answer["failedDecks"]); len(failed) != 1 || failed[0] != "Herds.md" {
		t.Errorf("failedDecks = %v, want [Herds.md]", answer["failedDecks"])
	}
}

func TestCardToolSchemasNameEveryKeyInCamelCase(t *testing.T) {
	s, _ := newSession(t, vault())
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if strings.Contains(name, "_") {
			t.Errorf("the schema of %s names the key %q, which is not camelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestCardToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	s, _ := newSession(t, vault())
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, s.registry, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}

func TestCardCurrentReturnsTheCardIDKey(t *testing.T) {
	s, _ := newSession(t, vault())
	s.toolContext = map[string]string{
		"md.numen.views.flashcards.studio/deck":     "Rivers.md",
		"md.numen.views.flashcards.studio/card":     "w4r8t2mnqz",
		"md.numen.views.flashcards.studio/template": "Recall",
	}
	answer := call[map[string]any](t, s, "card_current", map[string]any{})
	requireKeys(t, "the answer of card_current", answer, []string{"deck", "cardId", "template"}, []string{"card"})
	if id, isText := answer["cardId"].(string); !isText || id != "w4r8t2mnqz" {
		t.Errorf("cardId = %v, want w4r8t2mnqz", answer["cardId"])
	}

	schema := getToolSchema(t, s.registry, "card_current", true)
	requireKeys(t, "the output schema of card_current", getSchemaProperties(t, schema), []string{"deck", "cardId", "template"}, []string{"card"})
}

func TestCardAddReturnsTheTotalKey(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_add", map[string]any{
		"path": "Animals.md", "cardModel": "Animal",
		"values":      []map[string]string{{"field": "Name", "text": "Guanaco"}},
		"fingerprint": deckFingerprint(t, s, "Animals.md"),
	})
	requireKeys(t, "the answer of card_add", answer, []string{"path", "fingerprint", "total", "cardId"}, []string{"cards"})
	if total, isNumber := answer["total"].(float64); !isNumber || total != 3 {
		t.Errorf("total = %v, want 3", answer["total"])
	}
}

func TestCardRemoveReturnsTheTotalKey(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_remove", map[string]any{
		"path": "Animals.md", "cardId": alpaca,
		"fingerprint": deckFingerprint(t, s, "Animals.md"),
	})
	requireKeys(t, "the answer of card_remove", answer, []string{"path", "fingerprint", "total"}, []string{"cards"})
	if total, isNumber := answer["total"].(float64); !isNumber || total != 1 {
		t.Errorf("total = %v, want 1", answer["total"])
	}
}

func TestCardWriteToolSchemasNameTheTotalKey(t *testing.T) {
	s, _ := newSession(t, vault())
	for _, name := range []string{
		"card_add", "card_edit", "card_value_remove", "card_remove",
		"card_section_add", "card_section_rename", "card_section_remove", "card_model_edit",
	} {
		schema := getToolSchema(t, s.registry, name, true)
		requireKeys(t, "the output schema of "+name, getSchemaProperties(t, schema), []string{"path", "fingerprint", "total"}, []string{"cards"})
	}
}

func TestCardToolInstructionsContainNoNarrativeMarker(t *testing.T) {
	s, _ := newSession(t, vault())
	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range getNarrativeMarkers() {
		if marker.MatchString(listing.Instructions) {
			t.Errorf("the instructions match %s: %q", marker, listing.Instructions)
		}
	}
}
