package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	transcription "github.com/jiva-studio/numen/plugins/engines-transcription"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

// The recogniser opens with the model the settings name.
func TestTheRecogniserOpensWithTheModelOfTheSettings(t *testing.T) {
	for _, tc := range []struct {
		values map[string]string
		model  string
		repo   string
	}{
		{map[string]string{}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8"},
		{map[string]string{"model": "parakeet-tdt-0.6b-v3-int8"}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8"},
		{map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"}, "parakeet-tdt-0.6b-v2-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8"},
		{map[string]string{"model": "whisper-large"}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8"},
	} {
		var got engine.Config
		model := &models{cfg: NewConfig("", ""), open: func(_ context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
			got = cfg
			return &heardModel{}, nil
		}}
		model.updateConfig(tc.values)
		if _, err := model.get(t.Context()); err != nil {
			t.Fatal(err)
		}
		if got.Model.Name != tc.model || !strings.Contains(got.Model.Repo, tc.repo) {
			t.Errorf("settings %v: got %s from %s, want %s from %s",
				tc.values, got.Model.Name, got.Model.Repo, tc.model, tc.repo)
		}
	}
}

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
