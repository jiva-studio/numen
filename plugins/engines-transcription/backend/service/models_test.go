package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	transcription "github.com/jiva-studio/numen/plugins/engines-transcription"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

// The recogniser opens with the model and the language the settings name.
func TestTheRecogniserOpensWithTheModelAndLanguageOfTheSettings(t *testing.T) {
	for _, tc := range []struct {
		values   map[string]string
		model    string
		repo     string
		language string
	}{
		{map[string]string{}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8", ""},
		{map[string]string{"model": "parakeet-tdt-0.6b-v3-int8", "language": "auto"}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8", "auto"},
		{map[string]string{"model": "parakeet-tdt-0.6b-v2-int8", "language": "en"}, "parakeet-tdt-0.6b-v2-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v2-int8", "en"},
		{map[string]string{"model": "whisper-large"}, "parakeet-tdt-0.6b-v3-int8", "sherpa-onnx-nemo-parakeet-tdt-0.6b-v3-int8", ""},
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
		if got.Model.Name != tc.model || !strings.Contains(got.Model.Repo, tc.repo) || got.Language != tc.language {
			t.Errorf("settings %v: got %s from %s in %q, want %s from %s in %q",
				tc.values, got.Model.Name, got.Model.Repo, got.Language, tc.model, tc.repo, tc.language)
		}
	}
}

// Each model the manifest offers is one the engine provisions, with the languages the manifest
// lists for it.
func TestEveryModelOfTheManifestIsOneTheEngineRuns(t *testing.T) {
	options := readManifestOptions(t)
	if len(options.models) == 0 {
		t.Fatal("the manifest offers no model")
	}
	for _, name := range options.models {
		model, ok := engine.FindModel(name)
		if !ok {
			t.Errorf("the manifest offers %s, which the engine does not run", name)
			continue
		}
		languages := options.languages[name]
		if len(languages) != len(model.Languages) {
			t.Errorf("%s: the manifest lists %v and the engine %v", name, languages, model.Languages)
			continue
		}
		for place, one := range model.Languages {
			if languages[place] != one {
				t.Errorf("%s: the manifest lists %v and the engine %v", name, languages, model.Languages)
			}
		}
	}
}

// manifestOptions are the models the manifest offers and the languages it lists under each.
type manifestOptions struct {
	models    []string
	languages map[string][]string
}

func readManifestOptions(t *testing.T) manifestOptions {
	t.Helper()
	type option struct {
		Value string `json:"value"`
	}
	var manifest struct {
		Settings []struct {
			Key       string   `json:"key"`
			Options   []option `json:"options"`
			OptionsBy struct {
				Options map[string][]option `json:"options"`
			} `json:"optionsBy"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(transcription.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	read := manifestOptions{languages: map[string][]string{}}
	for _, setting := range manifest.Settings {
		switch setting.Key {
		case "model":
			for _, one := range setting.Options {
				read.models = append(read.models, one.Value)
			}
		case "language":
			for model, list := range setting.OptionsBy.Options {
				for _, one := range list {
					read.languages[model] = append(read.languages[model], one.Value)
				}
			}
		}
	}
	return read
}
