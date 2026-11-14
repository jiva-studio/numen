package ocr

import (
	"encoding/json"
	"slices"
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
