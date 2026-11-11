package service

import (
	"encoding/json"
	"testing"

	transcription "github.com/jiva-studio/numen/plugins/engines-transcription"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

// Each model the manifest offers is one the engine provisions.
func TestEveryModelOfTheManifestIsOneTheEngineRuns(t *testing.T) {
	models := readManifestModels(t)
	if len(models) == 0 {
		t.Fatal("the manifest offers no model")
	}
	for _, name := range models {
		if _, ok := engine.FindModel(name); !ok {
			t.Errorf("the manifest offers %s, which the engine does not run", name)
		}
	}
}

func readManifestModels(t *testing.T) []string {
	t.Helper()
	type option struct {
		Value string `json:"value"`
	}
	var manifest struct {
		Settings []struct {
			Key     string   `json:"key"`
			Options []option `json:"options"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(transcription.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var models []string
	for _, setting := range manifest.Settings {
		if setting.Key == "model" {
			for _, one := range setting.Options {
				models = append(models, one.Value)
			}
		}
	}
	return models
}
