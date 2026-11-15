package taxonomy

import (
	"encoding/json"
	"slices"
	"testing"
)

type declaredCommand struct {
	ID   string `json:"id"`
	When string `json:"when"`
}

type manifestShape struct {
	Permissions  []string `json:"permissions"`
	Capabilities []string `json:"capabilities"`
	Contributes  struct {
		Commands []declaredCommand `json:"commands"`
		Triggers []struct {
			Event   string `json:"event"`
			Command string `json:"command"`
			When    string `json:"when"`
		} `json:"triggers"`
	} `json:"contributes"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestTheManifestUpdatesTheTagsOfANoteWhoseFrontmatterChanged(t *testing.T) {
	shape := readManifest(t)

	if len(shape.Contributes.Triggers) != 1 {
		t.Fatalf("triggers = %+v, want one", shape.Contributes.Triggers)
	}
	trigger := shape.Contributes.Triggers[0]
	if trigger.Event != "note.frontmatterChanged" || trigger.Command != "tags.update" || trigger.When != "target.isNote" {
		t.Errorf("trigger = %+v", trigger)
	}
	if !slices.ContainsFunc(shape.Contributes.Commands, func(c declaredCommand) bool { return c.ID == "tags.update" }) {
		t.Error("the manifest declares no command tags.update")
	}
}

func TestTheManifestReadsTheVaultAndServesTools(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Permissions, []string{"vault:read"}) || !slices.Equal(shape.Capabilities, []string{"tools"}) {
		t.Errorf("permissions %v and capabilities %v, want vault:read and tools", shape.Permissions, shape.Capabilities)
	}
}
