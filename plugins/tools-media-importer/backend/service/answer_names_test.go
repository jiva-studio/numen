package service

import (
	"encoding/json"
	"fmt"
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

// getNarrativeMarkers returns the patterns of narrative wording that tool descriptions, field
// descriptions and tool instructions must not contain.
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

// getImportAnswer calls url_import on Mills with the raw arguments and returns its answer as a map.
func (s *importScene) getImportAnswer(t *testing.T, arguments string) map[string]any {
	t.Helper()
	res := invokeAtOnce(t, s.registry, tools.Call{VaultID: "Mills", Name: "url_import", Arguments: []byte(arguments)})
	if res.GetResult().GetIsError() {
		t.Fatalf("url_import %s was refused: %s", arguments, res.GetResult().GetText())
	}
	return decodeAnswer(t, res.GetResult().GetStructured())
}

func TestURLImportAnswersHasTextTrueWhenTheTextWasFetched(t *testing.T) {
	s := newReleasedScene(t)
	answer := s.getImportAnswer(t, `{"url":"`+articleAddress+`"}`)
	requireKeys(t, "the answer of url_import", answer, []string{"hasText"}, []string{"nothing"})
	if answer["hasText"] != true {
		t.Errorf("hasText = %v, want true for a fetched article", answer["hasText"])
	}
}

func TestURLImportAnswersHasTextFalseWhenTheAddressPublishesNoText(t *testing.T) {
	s := newReleasedScene(t)
	answer := s.getImportAnswer(t, `{"url":"`+silentAddress+`"}`)
	requireKeys(t, "the answer of url_import", answer, []string{"hasText"}, []string{"nothing"})
	if answer["hasText"] != false {
		t.Errorf("hasText = %v, want false for an address with no text", answer["hasText"])
	}
}

func TestURLImportDownloadsTheVideoWhenDownloadVideoIsSet(t *testing.T) {
	s := newReleasedScene(t)
	answer := s.getImportAnswer(t, `{"url":"`+videoAddress+`","downloadVideo":true}`)
	s.links.Wait()
	requireKeys(t, "the answer of url_import", answer, []string{"downloadTaskId"}, []string{"copy_task_id", "copy_error"})
	if want := "copy:" + fmt.Sprint(answer["path"]); answer["downloadTaskId"] != want {
		t.Errorf("downloadTaskId = %v, want %s", answer["downloadTaskId"], want)
	}
	if !hasCopy(t, s.store) {
		t.Error("the video was not downloaded with downloadVideo set")
	}
}

func TestURLImportRefusesTheCopyArgumentAndStartsNoDownload(t *testing.T) {
	s := newReleasedScene(t)
	res := invokeAtOnce(t, s.registry, tools.Call{
		VaultID: "Mills", Name: "url_import", Arguments: []byte(`{"url":"` + videoAddress + `","copy":true}`),
	})
	s.links.Wait()
	if !res.GetResult().GetIsError() || !strings.Contains(res.GetResult().GetText(), "copy") {
		t.Errorf("url_import with the argument copy was answered %+v, want an error naming copy", res.GetResult())
	}
	if hasCopy(t, s.store) {
		t.Error("the video was downloaded for the argument copy")
	}
}

func TestURLImportReturnsTheDownloadErrorKeyForAnAddressWithNoVideo(t *testing.T) {
	s := newReleasedScene(t)
	answer := s.getImportAnswer(t, `{"url":"`+articleAddress+`","downloadVideo":true}`)
	s.links.Wait()
	requireKeys(t, "the answer of url_import", answer, []string{"downloadError"}, []string{"copy_error", "copy_task_id"})
	if taskID, present := answer["downloadTaskId"]; present && taskID != "" {
		t.Errorf("downloadTaskId = %v, want none for an address with no video", taskID)
	}
	if answer["downloadError"] == "" {
		t.Error("downloadError is empty for an address with no video")
	}
}

func TestURLImportSchemasNameTheDownloadAndTextKeys(t *testing.T) {
	s := newReleasedScene(t)
	input := getSchemaProperties(t, getToolSchema(t, s.registry, "url_import", false))
	requireKeys(t, "the input schema of url_import", input, []string{"url", "folder", "downloadVideo"}, []string{"copy"})
	output := getSchemaProperties(t, getToolSchema(t, s.registry, "url_import", true))
	requireKeys(t, "the output schema of url_import", output,
		[]string{"hasText", "downloadTaskId", "downloadError"}, []string{"nothing", "copy_task_id", "copy_error"})
}

func TestURLImportInstructionsAndDescriptionNameDownloadVideo(t *testing.T) {
	s := newReleasedScene(t)
	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	instructions := strings.Join(getToolSentences(listing, "url_import"), " ")
	if !strings.Contains(instructions, "downloadVideo") || strings.Contains(instructions, "Set copy") {
		t.Errorf("the instructions do not name downloadVideo in place of copy: %q", instructions)
	}
	for _, info := range listing.Tools {
		if description := info.GetDescription(); !strings.Contains(description, "downloadVideo") || strings.Contains(description, "copy set") {
			t.Errorf("the description of %s does not name downloadVideo in place of copy: %q", info.GetName(), description)
		}
	}
}

func TestMediaImporterToolSchemasNameEveryKeyInCamelCase(t *testing.T) {
	s := newReleasedScene(t)
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if strings.Contains(name, "_") {
			t.Errorf("the schema of %s names the key %q, which is not camelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestMediaImporterToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	s := newReleasedScene(t)
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, s.registry, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}

func TestMediaImporterToolInstructionsContainNoNarrativeMarker(t *testing.T) {
	s := newReleasedScene(t)
	listing, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(getToolSentences(listing, "url_import")) == 0 {
		t.Fatal("url_import lists no instructions")
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

// getToolSentences is the instruction sentences the tool of the name lists.
func getToolSentences(listing tools.Listing, name string) []string {
	for _, info := range listing.Tools {
		if info.GetName() == name {
			return info.Instructions
		}
	}
	return nil
}

// isOneSentence reports whether the text is one sentence: it ends with a full stop and holds no
// other end of a sentence.
func isOneSentence(text string) bool {
	return strings.TrimSpace(text) == text && strings.HasSuffix(text, ".") && !regexp.MustCompile(`[.!?]\s+\S`).MatchString(text)
}

// lowerCamelCase is the form of every property name of a tool schema.
var lowerCamelCase = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

func TestMediaImporterToolSchemasNameEveryPropertyInLowerCamelCase(t *testing.T) {
	s := newReleasedScene(t)
	visitEveryToolSchema(t, s.registry, func(tool, name string) {
		if !lowerCamelCase.MatchString(name) {
			t.Errorf("the schema of %s names the property %q, which is not lowerCamelCase", tool, name)
		}
	}, func(string, string) {})
}
