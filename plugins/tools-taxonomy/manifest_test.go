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
	if strings.Contains(string(shape.Contributes), "triggers") {
		t.Errorf("contributes = %s, want no triggers", shape.Contributes)
	}
}

func TestTheManifestReadsAndWritesTheVaultAndServesTools(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Permissions, []string{"vault:read", "vault:write"}) || !slices.Equal(shape.Capabilities, []string{"tools"}) {
		t.Errorf("permissions %v and capabilities %v, want vault:read, vault:write and tools", shape.Permissions, shape.Capabilities)
	}
}

type contributedCommand struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Icon  string `json:"icon"`
	When  string `json:"when"`
}

type contributedSlot struct {
	Slot    string `json:"slot"`
	Command string `json:"command"`
	Group   string `json:"group"`
	Order   int    `json:"order"`
	When    string `json:"when"`
}

func readContributions(t *testing.T) (commands []contributedCommand, slots []contributedSlot) {
	t.Helper()
	var contributes struct {
		Commands []contributedCommand `json:"commands"`
		Slots    []contributedSlot    `json:"slots"`
	}
	if err := json.Unmarshal(readManifest(t).Contributes, &contributes); err != nil {
		t.Fatal(err)
	}
	return contributes.Commands, contributes.Slots
}

func findSetTagsEntry(slots []contributedSlot, slot string) (contributedSlot, bool) {
	for _, one := range slots {
		if one.Slot == slot && one.Command == "tags.set" {
			return one, true
		}
	}
	return contributedSlot{}, false
}

func TestTheManifestDeclaresTheCommandSetTagsOverNotes(t *testing.T) {
	commands, _ := readContributions(t)

	want := contributedCommand{ID: "tags.set", Title: "Set tags", Icon: "tag", When: "target.isNote"}
	if !slices.ContainsFunc(commands, func(one contributedCommand) bool { return one == want }) {
		t.Errorf("commands = %+v, want %+v", commands, want)
	}
}

func TestTheManifestOffersSetTagsInTheEditGroupOfTheFileTreeMenuOverNotes(t *testing.T) {
	_, slots := readContributions(t)

	got, ok := findSetTagsEntry(slots, "fileTree.context")

	if !ok || got.Group != "2-edit" || got.Order != 15 || got.When != "target.isNote" {
		t.Errorf("fileTree.context entry = %+v (found %v), want group 2-edit, order 15, when target.isNote", got, ok)
	}
}

func TestTheManifestOffersSetTagsInThePlexGroupOfThePlexNodeMenu(t *testing.T) {
	_, slots := readContributions(t)

	got, ok := findSetTagsEntry(slots, "plexGraph.context")

	if !ok || got.Group != "3-plex" || got.Order != 50 || got.When != "" {
		t.Errorf("plexGraph.context entry = %+v (found %v), want group 3-plex, order 50, with no condition", got, ok)
	}
}

func TestTheManifestPlacesTheTagDotsInTheRowsOfTheFileTree(t *testing.T) {
	var shape struct {
		Frontend *json.RawMessage `json:"frontend"`
		Services []string         `json:"services"`
		Contrib  struct {
			Slots []struct {
				Slot string `json:"slot"`
				View string `json:"view"`
			} `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}

	if shape.Frontend == nil {
		t.Error("the manifest declares no frontend")
	}
	if !slices.Equal(shape.Services, []string{"md.numen.tools.taxonomy"}) {
		t.Errorf("services = %v, want [md.numen.tools.taxonomy]", shape.Services)
	}
	var placed []string
	for _, one := range shape.Contrib.Slots {
		placed = append(placed, one.Slot+" "+one.View)
	}
	if !slices.Contains(placed, "fileTree.rowEnd file-tags") {
		t.Errorf("slots = %v, want the view file-tags in fileTree.rowEnd", placed)
	}
}

func TestTheManifestPlacesTheTagDotsInTheRowsOfTheFileTree(t *testing.T) {
	var shape struct {
		Frontend *json.RawMessage `json:"frontend"`
		Services []string         `json:"services"`
		Contrib  struct {
			Slots []struct {
				Slot string `json:"slot"`
				View string `json:"view"`
			} `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}

	if shape.Frontend == nil {
		t.Error("the manifest declares no frontend")
	}
	if !slices.Equal(shape.Services, []string{"md.numen.tools.taxonomy"}) {
		t.Errorf("services = %v, want [md.numen.tools.taxonomy]", shape.Services)
	}
	var placed []string
	for _, one := range shape.Contrib.Slots {
		placed = append(placed, one.Slot+" "+one.View)
	}
	if !slices.Contains(placed, "fileTree.rowEnd file-tags") {
		t.Errorf("slots = %v, want the view file-tags in fileTree.rowEnd", placed)
	}
}

func TestTheManifestHasThePlexTagsViewInTheNodeBelowSlot(t *testing.T) {
	var shape struct {
		Contrib struct {
			Slots []struct {
				Slot string `json:"slot"`
				View string `json:"view"`
			} `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}

	var placed []string
	for _, one := range shape.Contrib.Slots {
		placed = append(placed, one.Slot+" "+one.View)
	}
	if !slices.Contains(placed, "plexGraph.nodeBelow plex-tags") {
		t.Errorf("slots = %v, want the view plex-tags in plexGraph.nodeBelow", placed)
	}
	if slices.Contains(placed, "plexGraph.nodeAbove plex-tags") {
		t.Errorf("slots = %v, want the tags under the node and not over it", placed)
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
