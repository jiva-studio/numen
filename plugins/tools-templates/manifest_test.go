package templates

import (
	"encoding/json"
	"slices"
	"testing"
)

type manifestShape struct {
	Targets     []string `json:"targets"`
	Permissions []string `json:"permissions"`
	Services    []string `json:"services"`
	Contributes struct {
		Commands []struct {
			ID        string            `json:"id"`
			Title     string            `json:"title"`
			Arguments []json.RawMessage `json:"arguments"`
			When      string            `json:"when"`
		} `json:"commands"`
		Slots []struct {
			Slot    string `json:"slot"`
			Command string `json:"command"`
			When    string `json:"when"`
		} `json:"slots"`
	} `json:"contributes"`
	Settings []struct {
		Key     string `json:"key"`
		Type    string `json:"type"`
		Default string `json:"default"`
	} `json:"settings"`
	Dependencies []json.RawMessage `json:"dependencies"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestTheManifestDeclaresTheCommandTheFolderSettingTheFolderEntryAndTheService(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", shape.Targets)
	}
	if !slices.Equal(shape.Permissions, []string{"vault:read", "vault:write"}) {
		t.Errorf("permissions = %v", shape.Permissions)
	}
	if !slices.Equal(shape.Services, []string{"md.numen.tools.templates"}) {
		t.Errorf("services = %v", shape.Services)
	}
	commands := shape.Contributes.Commands
	if len(commands) != 1 || commands[0].ID != "note.createFromTemplate" || commands[0].Title != "New note from template" {
		t.Fatalf("commands = %+v", commands)
	}
	if len(commands[0].Arguments) != 0 || commands[0].When != "" {
		t.Errorf("the command has arguments or a condition: %+v", commands[0])
	}
	if len(shape.Settings) != 1 || shape.Settings[0].Key != "folder" || shape.Settings[0].Type != "string" || shape.Settings[0].Default != "Templates" {
		t.Errorf("settings = %+v", shape.Settings)
	}
	slots := shape.Contributes.Slots
	if len(slots) != 1 || slots[0].Slot != "fileTree.new" || slots[0].Command != "note.createFromTemplate" || slots[0].When != "target.isFolder" {
		t.Errorf("slots = %+v", slots)
	}
}

func TestTheManifestDeclaresNoProgramDependency(t *testing.T) {
	if got := readManifest(t).Dependencies; len(got) != 0 {
		t.Errorf("dependencies = %s", got)
	}
}
