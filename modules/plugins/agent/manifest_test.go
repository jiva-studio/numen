package agent

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/harness"
)

// The model setting is offered the models of the program the plugin runs.
func TestTheModelSettingIsOfferedTheModelsOfTheProgramThePluginRuns(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key      string `json:"key"`
			ModelsOf string `json:"modelsOf"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Settings {
		if one.Key == "model" {
			if one.ModelsOf != harness.ProgramName {
				t.Errorf("the model setting is offered the models of %q, want %q", one.ModelsOf, harness.ProgramName)
			}
			return
		}
	}
	t.Error("the plugin declares no model setting")
}
