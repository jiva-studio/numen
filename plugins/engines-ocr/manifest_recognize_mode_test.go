package ocr

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

type manifestSetting struct {
	Key     string `json:"key"`
	Type    string `json:"type"`
	Default any    `json:"default"`
	Options []struct {
		Value string `json:"value"`
	} `json:"options"`
}

type manifestTrigger struct {
	Event   string `json:"event"`
	Command string `json:"command"`
	When    string `json:"when"`
}

func readManifestForMode(t *testing.T) (settings []manifestSetting, triggers []manifestTrigger) {
	t.Helper()
	var manifest struct {
		Settings    []manifestSetting `json:"settings"`
		Contributes struct {
			Triggers []manifestTrigger `json:"triggers"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Settings, manifest.Contributes.Triggers
}

func TestTheManifestOffersRecognizeModeAsAnEnumOfThreeValuesDefaultingToOff(t *testing.T) {
	settings, _ := readManifestForMode(t)

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
	settings, _ := readManifestForMode(t)

	if slices.ContainsFunc(settings, func(s manifestSetting) bool { return s.Key == "recognizeAutomatically" }) {
		t.Error("the manifest still offers recognizeAutomatically")
	}
}

func getRecognizeCondition(t *testing.T) when.Condition {
	t.Helper()
	_, triggers := readManifestForMode(t)
	index := slices.IndexFunc(triggers, func(trigger manifestTrigger) bool { return trigger.Command == "app:file.createText" })
	if index < 0 {
		t.Fatal("the manifest has no trigger of app:file.createText")
	}
	condition, err := when.Parse(triggers[index].When)
	if err != nil {
		t.Fatalf("the trigger condition %q: %v", triggers[index].When, err)
	}
	return condition
}

func getTriggerValues(contentType, mode string, versions []any) map[string]any {
	return map[string]any{
		"target.contentType": contentType,
		"target.versions":    versions,
		"config": map[string]any{"md": map[string]any{"numen": map[string]any{"engines": map[string]any{
			"ocr": map[string]any{"recognizeMode": mode},
		}}}},
	}
}

func TestTheTriggerFiresForAPDFWhenTheModeIsWhenNoTextLayerOrAlways(t *testing.T) {
	condition := getRecognizeCondition(t)

	for _, mode := range []string{"whenNoTextLayer", "always"} {
		if got := condition.Evaluate(getTriggerValues("application/pdf", mode, []any{})); got != when.True {
			t.Errorf("mode %s: the trigger gave %v for a PDF, want it to fire", mode, got)
		}
	}
}

func TestTheTriggerFiresForAnImageWhenTheModeIsWhenNoTextLayer(t *testing.T) {
	condition := getRecognizeCondition(t)

	if got := condition.Evaluate(getTriggerValues("image/png", "whenNoTextLayer", []any{})); got != when.True {
		t.Errorf("the trigger gave %v for a PNG, want it to fire", got)
	}
}

func TestTheTriggerDoesNotFireWhenTheModeIsOff(t *testing.T) {
	condition := getRecognizeCondition(t)

	if got := condition.Evaluate(getTriggerValues("application/pdf", "off", []any{})); got != when.False {
		t.Errorf("the trigger gave %v for a PDF under off, want it not to fire", got)
	}
}

func TestTheTriggerDoesNotFireForAFileWithAValidRecognition(t *testing.T) {
	condition := getRecognizeCondition(t)
	recognized := []any{map[string]any{"kind": "recognition", "pluginId": "md.numen.engines.ocr", "isLive": true, "isValid": true}}

	for _, mode := range []string{"whenNoTextLayer", "always"} {
		if got := condition.Evaluate(getTriggerValues("application/pdf", mode, recognized)); got != when.False {
			t.Errorf("mode %s: the trigger gave %v for a recognized PDF, want it not to fire", mode, got)
		}
	}
}

func TestTheTriggerFiresAgainForAFileWhoseRecognitionIsNotValid(t *testing.T) {
	condition := getRecognizeCondition(t)
	stale := []any{map[string]any{"kind": "recognition", "pluginId": "md.numen.engines.ocr", "isLive": false, "isValid": false}}

	if got := condition.Evaluate(getTriggerValues("application/pdf", "always", stale)); got != when.True {
		t.Errorf("the trigger gave %v for a PDF whose recognition is stale, want it to fire", got)
	}
}

func TestTheTriggerDoesNotFireForAFileThatIsNoPDFOrImage(t *testing.T) {
	condition := getRecognizeCondition(t)

	if got := condition.Evaluate(getTriggerValues("text/markdown", "always", []any{})); got != when.False {
		t.Errorf("the trigger gave %v for a text file, want it not to fire", got)
	}
}
