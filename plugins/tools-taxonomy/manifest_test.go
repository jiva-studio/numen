package taxonomy

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
)

type manifestShape struct {
	Permissions      []string        `json:"permissions"`
	Capabilities     []string        `json:"capabilities"`
	ActivationEvents []string        `json:"activationEvents"`
	Contributes      json.RawMessage `json:"contributes"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestTheManifestUpdatesTheTagsOnTheFilesChangedEventForNotesAlone(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.ActivationEvents, []string{"onFilesChanged:text/markdown"}) {
		t.Errorf("activationEvents = %v, want [onFilesChanged:text/markdown]", shape.ActivationEvents)
	}
	if strings.Contains(string(shape.Contributes), "triggers") || strings.Contains(string(shape.Contributes), "commands") {
		t.Errorf("contributes = %s, want no triggers and no commands", shape.Contributes)
	}
}

func TestTheManifestReadsTheVaultAndServesTools(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Permissions, []string{"vault:read"}) || !slices.Equal(shape.Capabilities, []string{"tools"}) {
		t.Errorf("permissions %v and capabilities %v, want vault:read and tools", shape.Permissions, shape.Capabilities)
	}
}

type searchFunction struct {
	Name        string `json:"name"`
	Level       string `json:"level"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Args        []struct {
		Type string `json:"type"`
	} `json:"args"`
}

func readSearchFunctions(t *testing.T) map[string]searchFunction {
	t.Helper()
	var contributes struct {
		SearchFunctions struct {
			Functions []searchFunction `json:"functions"`
		} `json:"searchFunctions"`
	}
	if err := json.Unmarshal(readManifest(t).Contributes, &contributes); err != nil {
		t.Fatal(err)
	}
	byName := map[string]searchFunction{}
	for _, function := range contributes.SearchFunctions.Functions {
		byName[function.Name] = function
	}
	return byName
}

func TestTheManifestDeclaresHasTagAndHasTagExactlyWithOneStringArgumentAndNoTags(t *testing.T) {
	functions := readSearchFunctions(t)

	for _, name := range []string{"hasTag", "hasTagExactly"} {
		function, ok := functions[name]
		if !ok {
			t.Errorf("the manifest declares %v, want %s", slices.Sorted(maps.Keys(functions)), name)
			continue
		}
		if function.Level != "document" || len(function.Args) != 1 || function.Args[0].Type != "string" || function.Description == "" || !strings.Contains(function.Example, name+"(") {
			t.Errorf("%s = %+v, want level document, one string argument, a description and an example", name, function)
		}
	}
	if _, ok := functions["tags"]; ok {
		t.Error("the manifest declares tags, want it renamed to hasTag")
	}
}

func TestTheManifestDeclaresHasAnyTagWithoutArgumentsAndNoIsTagged(t *testing.T) {
	functions := readSearchFunctions(t)

	function, ok := functions["hasAnyTag"]
	if !ok {
		t.Fatalf("the manifest declares %v, want hasAnyTag", slices.Sorted(maps.Keys(functions)))
	}
	if function.Level != "document" || len(function.Args) != 0 || function.Description == "" || function.Example != `!hasAnyTag() && kind == "note"` {
		t.Errorf("hasAnyTag = %+v, want level document, no argument, a description and the example", function)
	}
	if _, ok := functions["isTagged"]; ok {
		t.Error("the manifest declares isTagged, want it renamed to hasAnyTag")
	}
	if !slices.Equal(slices.Sorted(maps.Keys(functions)), []string{"hasAnyTag", "hasTag", "hasTagExactly"}) {
		t.Errorf("the manifest declares %v, want hasAnyTag, hasTag and hasTagExactly", slices.Sorted(maps.Keys(functions)))
	}
}
