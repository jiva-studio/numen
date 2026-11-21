package viewstodos

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
		TabKinds []json.RawMessage `json:"tabKinds"`
		Commands []json.RawMessage `json:"commands"`
		Slots    []struct {
			Slot string `json:"slot"`
			View string `json:"view"`
			When string `json:"when"`
		} `json:"slots"`
	} `json:"contributes"`
	Settings []struct {
		Key string `json:"key"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func TestTheManifestDeclaresTheEditorTargetTheReadAndWritePermissionsAndTheService(t *testing.T) {
	shape := readManifest(t)

	if !slices.Equal(shape.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", shape.Targets)
	}
	if !slices.Equal(shape.Permissions, []string{"vault:read", "vault:write"}) {
		t.Errorf("permissions = %v", shape.Permissions)
	}
	if !slices.Equal(shape.Services, []string{"md.numen.todos"}) {
		t.Errorf("services = %v", shape.Services)
	}
}

func TestTheManifestDeclaresTheStatusViewInTheTitleBarWhenAVaultIsOpen(t *testing.T) {
	slots := readManifest(t).Contributes.Slots

	if len(slots) != 1 || slots[0].Slot != "titleBar.status" || slots[0].View != "status" || slots[0].When != "window.isVaultOpen" {
		t.Errorf("slots = %+v", slots)
	}
}

func TestTheManifestDeclaresNoTabKindAndNoCommandBeforeTheTabExists(t *testing.T) {
	shape := readManifest(t)

	if len(shape.Contributes.TabKinds) != 0 || len(shape.Contributes.Commands) != 0 {
		t.Errorf("tabKinds = %s, commands = %s", shape.Contributes.TabKinds, shape.Contributes.Commands)
	}
}

func TestTheManifestDeclaresTheIgnoredFoldersSettingOnly(t *testing.T) {
	settings := readManifest(t).Settings

	if len(settings) != 1 || settings[0].Key != "ignoredFolders" {
		t.Errorf("settings = %+v", settings)
	}
}
