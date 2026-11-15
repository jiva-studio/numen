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

func TestTheManifestUpdatesTheTagsOnTheFilesChangedEventAlone(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.ActivationEvents, []string{"onFilesChanged"}) {
		t.Errorf("activationEvents = %v, want [onFilesChanged]", shape.ActivationEvents)
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
