package service_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

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

func requireList(t *testing.T, what string, value any) []any {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is %v, not an array", what, value)
	}
	return list
}

func requireObject(t *testing.T, what string, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s is %v, not an object", what, value)
	}
	return object
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

func TestCardFieldRenameReturnsTheUpdatedDecksAndHeadings(t *testing.T) {
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

func TestCardCurrentGetReturnsPathCardIDAndTemplateName(t *testing.T) {
	s, _ := newSession(t, vault())
	s.toolContext = map[string]string{
		"md.numen.views.flashcards.studio/deck":     "Rivers.md",
		"md.numen.views.flashcards.studio/card":     "w4r8t2mnqz",
		"md.numen.views.flashcards.studio/template": "Recall",
	}
	answer := call[map[string]any](t, s, "card_current_get", map[string]any{})
	requireKeys(t, "the answer of card_current_get", answer, []string{"path", "cardId", "templateName"}, []string{"card", "deck", "template"})
	if id, isText := answer["cardId"].(string); !isText || id != "w4r8t2mnqz" {
		t.Errorf("cardId = %v, want w4r8t2mnqz", answer["cardId"])
	}
	if path, isText := answer["path"].(string); !isText || path != "Rivers.md" {
		t.Errorf("path = %v, want Rivers.md", answer["path"])
	}
	if name, isText := answer["templateName"].(string); !isText || name != "Recall" {
		t.Errorf("templateName = %v, want Recall", answer["templateName"])
	}

	schema := getToolSchema(t, s.registry, "card_current_get", true)
	requireKeys(t, "the output schema of card_current_get", getSchemaProperties(t, schema), []string{"path", "cardId", "templateName"}, []string{"card", "deck", "template"})
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

func getSectionedVault() map[string]string {
	return map[string]string{
		"Animal.md": animalCardModel,
		"Herd.md": "---\ntype: deck\n---\n\n" +
			"# Camelids\n\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n" +
			"# Bovids\n\n## Bison ^p4r7t2wxk9\n\n[[Animal]]\n\n### Name\n\nBison\n",
	}
}

func TestCardReadReturnsTheSectionIndexKey(t *testing.T) {
	s, _ := newSession(t, getSectionedVault())
	answer := call[map[string]any](t, s, "card_read", map[string]any{"path": "Herd.md"})
	cardList := requireList(t, "answer[cards]", answer["cards"])
	if len(cardList) != 2 {
		t.Fatalf("cards = %v, want two", answer["cards"])
	}
	bison := requireObject(t, "cards[1]", cardList[1])
	requireKeys(t, "a card of card_read", bison, []string{"sectionIndex"}, []string{"section"})
	if index, isNumber := bison["sectionIndex"].(float64); !isNumber || index != 1 {
		t.Errorf("sectionIndex = %v, want 1", bison["sectionIndex"])
	}

	schema := getToolSchema(t, s.registry, "card_read", true)
	requireKeys(t, "the output schema of a card_read card", getSchemaProperties(t, schema, "cards"), []string{"sectionIndex"}, []string{"section"})
}

func TestCardReadSkipsTheCardsBeforeTheOffset(t *testing.T) {
	s, _ := newSession(t, vault())
	read := readCards(t, s, map[string]any{"path": "Animals.md", "offset": 1, "limit": 1})
	if len(read.Cards) != 1 || read.Cards[0].CardID != alpaca {
		t.Errorf("card_read with offset 1 returned %+v, want the card %s", read.Cards, alpaca)
	}

	input := getSchemaProperties(t, getToolSchema(t, s.registry, "card_read", false))
	requireKeys(t, "the input schema of card_read", input, []string{"offset", "limit"}, []string{"from"})
}

func TestCardAddWritesTheCardIntoTheSectionIndex(t *testing.T) {
	s, v := newSession(t, getSectionedVault())
	call[map[string]any](t, s, "card_add", map[string]any{
		"path": "Herd.md", "cardModel": "Animal", "sectionIndex": 0,
		"values":      []map[string]string{{"field": "Name", "text": "Guanaco"}},
		"fingerprint": deckFingerprint(t, s, "Herd.md"),
	})
	content := readVaultFile(t, v, "Herd.md")
	guanaco, bovids := strings.Index(content, "## Guanaco"), strings.Index(content, "# Bovids")
	if guanaco < 0 || bovids < 0 || guanaco > bovids {
		t.Errorf("the card is not in section 0: %q", content)
	}

	input := getSchemaProperties(t, getToolSchema(t, s.registry, "card_add", false))
	requireKeys(t, "the input schema of card_add", input, []string{"sectionIndex"}, []string{"section"})
}

func TestCardSectionRenameRenamesTheSectionAtTheSectionIndex(t *testing.T) {
	s, v := newSession(t, getSectionedVault())
	call[map[string]any](t, s, "card_section_rename", map[string]any{
		"path": "Herd.md", "sectionIndex": 1, "name": "Cattle",
		"fingerprint": deckFingerprint(t, s, "Herd.md"),
	})
	content := readVaultFile(t, v, "Herd.md")
	if !strings.Contains(content, "# Camelids\n") || !strings.Contains(content, "# Cattle\n") || strings.Contains(content, "# Bovids") {
		t.Errorf("section 1 was not renamed to Cattle: %q", content)
	}

	input := getSchemaProperties(t, getToolSchema(t, s.registry, "card_section_rename", false))
	requireKeys(t, "the input schema of card_section_rename", input, []string{"sectionIndex"}, []string{"section"})
}

func TestCardSectionRemoveRemovesTheSectionAtTheSectionIndex(t *testing.T) {
	s, v := newSession(t, getSectionedVault())
	call[map[string]any](t, s, "card_section_remove", map[string]any{
		"path": "Herd.md", "sectionIndex": 1,
		"fingerprint": deckFingerprint(t, s, "Herd.md"),
	})
	content := readVaultFile(t, v, "Herd.md")
	if !strings.Contains(content, "# Camelids\n") || strings.Contains(content, "# Bovids") {
		t.Errorf("section 1 was not removed: %q", content)
	}

	input := getSchemaProperties(t, getToolSchema(t, s.registry, "card_section_remove", false))
	requireKeys(t, "the input schema of card_section_remove", input, []string{"sectionIndex"}, []string{"section"})
}

// lowerCamelCase is the form of every property name of a tool schema.
var lowerCamelCase = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestCardToolSchemasNameEveryPropertyInLowerCamelCase(t *testing.T) {
	s, _ := newSession(t, vault())
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if !lowerCamelCase.MatchString(name) {
			t.Errorf("the schema of %s names the property %q, which is not lowerCamelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestCardDeckCreateReturnsThePathIDAndTitleKeys(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_deck_create", map[string]any{"title": "Terms", "folder": "decks"})
	requireKeys(t, "the answer of card_deck_create", answer, []string{"path", "id", "title"}, []string{"Path", "ID", "Title"})
	if answer["path"] != "decks/Terms.md" || answer["title"] != "Terms" {
		t.Errorf("card_deck_create returned %v, want the path decks/Terms.md and the title Terms", answer)
	}
}

func TestCardModelCreateReturnsThePathIDAndTitleKeys(t *testing.T) {
	s, _ := newSession(t, vault())
	answer := call[map[string]any](t, s, "card_model_create", map[string]any{"title": "Term", "fields": []string{"Word", "Meaning"}, "templates": []any{}})
	requireKeys(t, "the answer of card_model_create", answer, []string{"path", "id", "title"}, []string{"Path", "ID", "Title"})
	if answer["path"] != "Term.md" || answer["title"] != "Term" {
		t.Errorf("card_model_create returned %v, want the path Term.md and the title Term", answer)
	}
}
