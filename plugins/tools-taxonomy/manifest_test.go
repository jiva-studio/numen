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
