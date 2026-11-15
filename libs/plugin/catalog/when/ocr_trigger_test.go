package when_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
)

// The trigger of the OCR plugin lives here and not in the plugin: a plugin depends on the plugin SDK
// alone, and this package evaluates the conditions the way the host does.
const ocrManifestPath = "../../../../plugins/engines-ocr/manifest.json"

type ocrTrigger struct {
	Command string `json:"command"`
	When    string `json:"when"`
}

func getOCRCondition(t *testing.T) when.Condition {
	t.Helper()
	text, err := os.ReadFile(ocrManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Contributes struct {
			Triggers []ocrTrigger `json:"triggers"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(text, &manifest); err != nil {
		t.Fatal(err)
	}
	triggers := manifest.Contributes.Triggers
	index := slices.IndexFunc(triggers, func(trigger ocrTrigger) bool { return trigger.Command == "app:file.createMissingText" })
	if index < 0 {
		t.Fatal("the manifest has no trigger of app:file.createMissingText")
	}
	return parse(t, triggers[index].When)
}

func getOCRTarget(contentType, mode string, versions []any) map[string]any {
	return map[string]any{
		"target.contentType": contentType,
		"target.versions":    versions,
		"config": map[string]any{"md": map[string]any{"numen": map[string]any{"engines": map[string]any{
			"ocr": map[string]any{"recognizeMode": mode},
		}}}},
	}
}

func TestTheOCRTriggerFiresForAPDFWhenTheModeIsWhenNoTextLayerOrAlways(t *testing.T) {
	condition := getOCRCondition(t)

	for _, mode := range []string{"whenNoTextLayer", "always"} {
		if got := condition.Evaluate(getOCRTarget("application/pdf", mode, []any{})); got != when.True {
			t.Errorf("mode %s: the trigger gave %v for a PDF, want it to fire", mode, got)
		}
	}
}

func TestTheOCRTriggerFiresForAnImageWhenTheModeIsWhenNoTextLayer(t *testing.T) {
	condition := getOCRCondition(t)

	if got := condition.Evaluate(getOCRTarget("image/png", "whenNoTextLayer", []any{})); got != when.True {
		t.Errorf("the trigger gave %v for a PNG, want it to fire", got)
	}
}

func TestTheOCRTriggerDoesNotFireWhenTheModeIsOff(t *testing.T) {
	condition := getOCRCondition(t)

	if got := condition.Evaluate(getOCRTarget("application/pdf", "off", []any{})); got != when.False {
		t.Errorf("the trigger gave %v for a PDF under off, want it not to fire", got)
	}
}

func TestTheOCRTriggerDoesNotFireForAFileWithAValidRecognition(t *testing.T) {
	condition := getOCRCondition(t)
	recognized := []any{map[string]any{"kind": "recognition", "pluginId": "md.numen.engines.ocr", "isLive": true, "isValid": true}}

	for _, mode := range []string{"whenNoTextLayer", "always"} {
		if got := condition.Evaluate(getOCRTarget("application/pdf", mode, recognized)); got != when.False {
			t.Errorf("mode %s: the trigger gave %v for a recognized PDF, want it not to fire", mode, got)
		}
	}
}

func TestTheOCRTriggerFiresAgainForAFileWhoseRecognitionIsNotValid(t *testing.T) {
	condition := getOCRCondition(t)
	stale := []any{map[string]any{"kind": "recognition", "pluginId": "md.numen.engines.ocr", "isLive": false, "isValid": false}}

	if got := condition.Evaluate(getOCRTarget("application/pdf", "always", stale)); got != when.True {
		t.Errorf("the trigger gave %v for a PDF whose recognition is stale, want it to fire", got)
	}
}

func TestTheOCRTriggerDoesNotFireForAFileThatIsNoPDFOrImage(t *testing.T) {
	condition := getOCRCondition(t)

	if got := condition.Evaluate(getOCRTarget("text/markdown", "always", []any{})); got != when.False {
		t.Errorf("the trigger gave %v for a text file, want it not to fire", got)
	}
}
