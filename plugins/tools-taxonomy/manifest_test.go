package taxonomy

import (
	"encoding/json"
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

	want := contributedCommand{ID: "tags.set", Title: "Set tags", When: "target.isNote"}
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
