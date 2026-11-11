package models

import (
	"context"
	"errors"
	"strings"
	"testing"

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
		model := NewHolder(engine.Defaults(), false, func(_ context.Context, cfg engine.Config) (engine.TranscriberPort, error) {
			opens++
			got = cfg
			return engine.NewMockTranscriber(), nil
		})
		model.UpdateSettings(tc.values)
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

	mockModels := NewHolder(engine.Defaults(), true, nil)
	mockPort, err := mockModels.get(t.Context())
	if err != nil || mockPort == nil {
		t.Fatalf("mock get() = %v, %v", mockPort, err)
	}

	errOpen := errors.New("open failed")
	failModels := NewHolder(engine.Defaults(), false, func(context.Context, engine.Config) (engine.TranscriberPort, error) {
		return nil, errOpen
	})
	if _, err := failModels.get(t.Context()); !errors.Is(err, errOpen) {
		t.Fatalf("expected errOpen, got %v", err)
	}
}

func TestTheRecognizerTranscribesThroughTheEngine(t *testing.T) {
	model := NewHolder(engine.Defaults(), true, nil)

	recognizer, err := model.GetRecognizer(t.Context())
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
