package service

import (
	"context"
	"encoding/json"
	"errors"
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
		var opens int
		model := &models{cfg: getConfig(), open: func(_ context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
			opens++
			got = cfg
			return engine.NewMockTranscriber(), nil
		}}
		model.updateConfig(tc.values)
		first, err := model.get(t.Context())
		if err != nil || first == nil {
			t.Fatalf("get() = %v, %v", first, err)
		}
		second, err := model.get(t.Context())
		if err != nil || second != first || opens != 1 {
			t.Fatalf("second get() = %v, %v (opens=%d)", second, err, opens)
		}
		if got.Model.Name != tc.model || !strings.Contains(got.Model.Repo, tc.repo) {
			t.Errorf("settings %v: got %s from %s, want %s from %s",
				tc.values, got.Model.Name, got.Model.Repo, tc.model, tc.repo)
		}
	}

	mockModels := &models{isMock: true}
	mockPort, err := mockModels.get(t.Context())
	if err != nil || mockPort == nil {
		t.Fatalf("mock get() = %v, %v", mockPort, err)
	}

	errOpen := errors.New("open failed")
	failModels := &models{open: func(context.Context, engine.Config) (engine.TranscriberPort, error) {
		return nil, errOpen
	}}
	if _, err := failModels.get(t.Context()); !errors.Is(err, errOpen) {
		t.Fatalf("expected errOpen, got %v", err)
	}
}

func TestGetConfigOverridesFromEnv(t *testing.T) {
	t.Setenv(engine.EnvDir, "")
	t.Setenv("NUMEN_ONNX_RUNTIME", "")
	base := getConfig()
	if base.Dir != "" || base.Runtime != "" {
		t.Fatalf("expected empty Dir and Runtime, got %+v", base)
	}

	t.Setenv(engine.EnvDir, "/env/models")
	t.Setenv("NUMEN_ONNX_RUNTIME", "/env/ort.so")
	fromEnv := getConfig()
	if fromEnv.Dir != "/env/models" || fromEnv.Runtime != "/env/ort.so" {
		t.Fatalf("expected env overrides, got %+v", fromEnv)
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

func TestTheRecognizerHearsThroughTheEngine(t *testing.T) {
	model := &models{isMock: true}

	recognizer, err := model.getRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	heard, err := recognizer.Open(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	segments, err := heard.GetSegments(t.Context(), 0, 16)
	if err != nil || len(segments) != 1 || segments[0].From != 0 || segments[0].To != 1000 {
		t.Fatalf("GetSegments = %+v, %v, want the one stretch the mock engine hears", segments, err)
	}
	words, err := recognizer.Transcribe(t.Context(), segments[0])
	if err != nil || words != "Mock transcribed text for automated test." {
		t.Errorf("Transcribe = %q, %v", words, err)
	}
	if err := heard.Close(); err != nil {
		t.Error(err)
	}
}
