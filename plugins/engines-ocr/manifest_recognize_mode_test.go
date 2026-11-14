package ocr

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

type manifestSetting struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Default any    `json:"default"`
	Options []struct {
		Value string `json:"value"`
	} `json:"options"`
}

func readManifestSettings(t *testing.T) []manifestSetting {
	t.Helper()
	var manifest struct {
		Settings []manifestSetting `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Settings
}

func TestTheManifestOffersRecognizeModeAsAnEnumOfThreeValuesDefaultingToOff(t *testing.T) {
	settings := readManifestSettings(t)

	index := slices.IndexFunc(settings, func(s manifestSetting) bool { return s.Key == "recognizeMode" })
	if index < 0 {
		t.Fatal("the manifest has no setting recognizeMode")
	}
	mode := settings[index]
	if mode.Type != "enum" || mode.Default != "off" {
		t.Errorf("recognizeMode is %q with default %v, want an enum with default off", mode.Type, mode.Default)
	}
	var values []string
	for _, option := range mode.Options {
		values = append(values, option.Value)
	}
	if want := []string{"off", "whenNoTextLayer", "always"}; !slices.Equal(values, want) {
		t.Errorf("recognizeMode offers %v, want %v", values, want)
	}
}

func TestTheManifestNoLongerOffersRecognizeAutomatically(t *testing.T) {
	settings := readManifestSettings(t)

	if slices.ContainsFunc(settings, func(s manifestSetting) bool { return s.Key == "recognizeAutomatically" }) {
		t.Error("the manifest still offers recognizeAutomatically")
	}
}

type manifestContributions struct {
	Slots []struct {
		Slot    string `json:"slot"`
		Command string `json:"command"`
	} `json:"slots"`
	Handlers []struct {
		Command string `json:"command"`
		When    string `json:"when"`
	} `json:"handlers"`
	Triggers []struct {
		Event   string `json:"event"`
		Command string `json:"command"`
		When    string `json:"when"`
	} `json:"triggers"`
}

func readManifestContributions(t *testing.T) manifestContributions {
	t.Helper()
	var manifest struct {
		Contributes manifestContributions `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Contributes
}

func TestTheManifestHandlesBothCreateTextAndCreateMissingText(t *testing.T) {
	contributes := readManifestContributions(t)

	for _, id := range []string{"app:file.createText", "app:file.createMissingText"} {
		if !hasHandler(contributes, id) {
			t.Errorf("the manifest has no handler of %s", id)
		}
	}
}

func TestTheFileTreeMenuRunsCreateText(t *testing.T) {
	contributes := readManifestContributions(t)

	var commands []string
	for _, slot := range contributes.Slots {
		if slot.Slot == "fileTree.context" {
			commands = append(commands, slot.Command)
		}
	}
	if want := []string{"app:file.createText"}; !slices.Equal(commands, want) {
		t.Errorf("the file tree menu runs %v, want %v", commands, want)
	}
}

func TestTheFileAddedTriggerRunsCreateMissingTextUnlessTheModeIsOff(t *testing.T) {
	contributes := readManifestContributions(t)

	if len(contributes.Triggers) != 1 {
		t.Fatalf("the manifest has %d triggers, want 1", len(contributes.Triggers))
	}
	trigger := contributes.Triggers[0]
	if trigger.Event != "file.added" || trigger.Command != "app:file.createMissingText" {
		t.Errorf("the trigger is %s of %s, want file.added of app:file.createMissingText", trigger.Event, trigger.Command)
	}
	for _, part := range []string{
		"config.md.numen.engines.ocr.recognizeMode != 'off'",
		"!target.versions.exists(v, v.kind == 'recognition' && v.isValid)",
	} {
		if !strings.Contains(trigger.When, part) {
			t.Errorf("the trigger condition %q lacks %q", trigger.When, part)
		}
	}
}

func hasHandler(contributes manifestContributions, command string) bool {
	for _, handler := range contributes.Handlers {
		if handler.Command == command {
			return true
		}
	}
	return false
}
