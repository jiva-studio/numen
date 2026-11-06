package service_test

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
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

func TestVaultListReturnsIsCurrentAndPathKeys(t *testing.T) {
	sky, registry := newSkyVault(t)
	garden := hosttest.New("garden-vault")
	garden.File("Beds.md", "# Beds\nRaised beds of oak.")
	sky.Beside(garden)

	var answer struct {
		Vaults []map[string]any `json:"vaults"`
	}
	decodeInto(t, callTool(t, registry, "garden-vault", "vault_list", `{}`), &answer)
	if len(answer.Vaults) != 2 {
		t.Fatalf("got %d vaults, want 2: %v", len(answer.Vaults), answer.Vaults)
	}
	for _, vault := range answer.Vaults {
		requireKeys(t, "a vault of vault_list", vault, []string{"id", "name", "path", "isCurrent"}, []string{"showing", "folder"})
		if want := vault["id"] == "garden-vault"; vault["isCurrent"] != want {
			t.Errorf("vault %v has isCurrent %v, want %v", vault["id"], vault["isCurrent"], want)
		}
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "vault_list", true), "vaults")
	requireKeys(t, "the output schema of a vault_list vault", properties, []string{"isCurrent", "path"}, []string{"showing", "folder"})
}

func TestVaultListDescriptionDoesNotNameTheShowingKey(t *testing.T) {
	_, registry := newSkyVault(t)
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, info := range listing.Tools {
		if info.GetName() == "vault_list" && strings.Contains(info.GetDescription(), "showing") {
			t.Errorf("the description of vault_list names showing: %s", info.GetDescription())
		}
	}
}

func TestVaultProblemsReturnsTheChecksKey(t *testing.T) {
	_, registry := newSkyVault(t)
	answer := decodeAnswer(t, callTool(t, registry, skyVault, "vault_problems", `{}`))
	requireKeys(t, "the answer of vault_problems", answer, []string{"problems", "checks"}, []string{"ran"})
	want := make([]any, 0, len(getDefaultChecks()))
	for _, check := range getDefaultChecks() {
		want = append(want, check)
	}
	if checks, _ := answer["checks"].([]any); !slices.Equal(checks, want) {
		t.Errorf("checks = %v, want %v", answer["checks"], want)
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "vault_problems", true))
	requireKeys(t, "the output schema of vault_problems", properties, []string{"checks"}, []string{"ran"})
}

func TestLibraryReadReturnsTheNextStartKey(t *testing.T) {
	l := newLibrary(t)
	res, err := l.calls.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_read", Arguments: []byte(`{"path":"` + guidePath + `","page":2}`)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		t.Fatal(res.GetResult().GetText())
	}
	answer := decodeAnswer(t, res.GetResult().GetStructured())
	requireKeys(t, "the answer of library_read", answer, []string{"nextStart"}, []string{"next"})
	if got, want := answer["nextStart"], float64(len(pageOne)+len(pageTwo)); got != want {
		t.Errorf("nextStart = %v, want %v", got, want)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.calls, "library_read", true))
	requireKeys(t, "the output schema of library_read", properties, []string{"nextStart"}, []string{"next"})
}

func TestLibrarySearchReturnsThePathKeyOfAMatch(t *testing.T) {
	l := newLibrary(t)
	res, err := l.calls.InvokeTool(t.Context(), tools.Call{VaultID: testVault, Name: "library_search", Arguments: []byte(`{"query":"heron"}`)})
	if err != nil {
		t.Fatal(err)
	}
	var answer struct {
		Matches []map[string]any `json:"matches"`
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &answer); err != nil {
		t.Fatal(err)
	}
	if len(answer.Matches) != 3 {
		t.Fatalf("got %d matches, want 3: %s", len(answer.Matches), res.GetResult().GetStructured())
	}
	requireKeys(t, "a match of library_search", answer.Matches[0], []string{"path"}, []string{"source"})
	if answer.Matches[0]["path"] != guidePath {
		t.Errorf("the first match has path %v, want %s", answer.Matches[0]["path"], guidePath)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.calls, "library_search", true), "matches")
	requireKeys(t, "the output schema of a library_search match", properties, []string{"path"}, []string{"source"})
}

func TestLibrarySearchAcceptsThePathPrefixArgument(t *testing.T) {
	l := newLibrary(t)
	var narrowed librarySearchAnswer
	if failure := l.call(t, "library_search", `{"query":"heron","pathPrefix":"recordings/"}`, &narrowed); failure != "" {
		t.Fatal(failure)
	}
	if got := getMatchPaths(narrowed); !slices.Equal(got, []string{talkPath}) {
		t.Errorf("the passages under recordings/ are %v, want only %s", got, talkPath)
	}

	properties := getSchemaProperties(t, getToolSchema(t, l.calls, "library_search", false))
	requireKeys(t, "the input schema of library_search", properties, []string{"pathPrefix"}, []string{"path_prefix"})
}

func TestFileCreateAnswersThePathAndSizeWithoutSuccess(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	answer := decodeAnswer(t, callTool(t, registry, testVault, "file_create", `{"path":"orbits.txt","content":"perihelion"}`))
	requireKeys(t, "the answer of file_create", answer, []string{"path", "size"}, []string{"success"})
	if answer["path"] != "orbits.txt" || answer["size"] != float64(len("perihelion")) {
		t.Errorf("file_create answered %v, want orbits.txt of 10 bytes", answer)
	}

	properties := getSchemaProperties(t, getToolSchema(t, registry, "file_create", true))
	requireKeys(t, "the output schema of file_create", properties, []string{"path", "size"}, []string{"success"})
}

func TestVaultToolSchemasNameEveryKeyInCamelCase(t *testing.T) {
	l := newLibrary(t)
	visitEveryToolSchema(t, l.calls, func(tool, name string) {
		if strings.Contains(name, "_") {
			t.Errorf("the schema of %s names the key %q, which is not camelCase", tool, name)
		}
	}, func(string, string) {})
}

func TestVaultToolDescriptionsContainNoNarrativeMarker(t *testing.T) {
	l := newLibrary(t)
	markers := getNarrativeMarkers()
	visitEveryToolSchema(t, l.calls, func(string, string) {}, func(tool, description string) {
		for _, marker := range markers {
			if marker.MatchString(description) {
				t.Errorf("a description of %s or its schema matches %s: %q", tool, marker, description)
			}
		}
	})
}

func TestVaultInstructionsNameTheKeysInCamelCase(t *testing.T) {
	_, svc := setupTestVault()
	registry, err := svc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	instructions := listing.Instructions
	for _, want := range []string{"`uri`", "kinds", "isCurrent"} {
		if !strings.Contains(instructions, want) {
			t.Errorf("the instructions do not name %q:\n%s", want, instructions)
		}
	}
	for _, old := range []string{"showing", "path_prefix", "next_offset", "next_start", "is_current", "note_count"} {
		if strings.Contains(instructions, old) {
			t.Errorf("the instructions name %q:\n%s", old, instructions)
		}
	}
}
