package service

import (
	"encoding/json"
	"slices"
	"testing"

	transcription "github.com/jiva-studio/numen/plugins/engines-transcription"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
)

// Each model the manifest offers is one the engine provisions, by the files its option names.
func TestEveryModelOfTheManifestIsOneTheEngineRuns(t *testing.T) {
	models := readManifestModels(t)
	if len(models) == 0 {
		t.Fatal("the manifest offers no model")
	}
	for name, dependencies := range models {
		model, ok := settings.FindModel(name)
		if !ok {
			t.Errorf("the manifest offers %s, which the engine does not run", name)
			continue
		}
		files := model.Dependencies
		if got := []string{files.Encoder, files.Decoder, files.Joiner, files.Tokens}; !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(dependencies))) {
			t.Errorf("%s fetches %v, its option names %v", name, got, dependencies)
		}
	}
}

func TestTheSegmenterIsAModelTheManifestDeclares(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID   string `json:"id"`
			Kind string `json:"kind"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(transcription.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	id := settings.NewDefaultConfig().Segmenter.Dependency
	for _, dependency := range manifest.Dependencies {
		if dependency.ID == id && dependency.Kind == "model" {
			return
		}
	}
	t.Errorf("the manifest declares no model %q", id)
}

func readManifestModels(t *testing.T) map[string][]string {
	t.Helper()
	type option struct {
		Value        string   `json:"value"`
		Dependencies []string `json:"dependencies"`
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
	models := map[string][]string{}
	for _, setting := range manifest.Settings {
		if setting.Key == "model" {
			for _, one := range setting.Options {
				models[one.Value] = one.Dependencies
			}
		}
	}
	return models
}
