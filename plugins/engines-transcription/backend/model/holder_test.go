package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// quietModel transcribes nothing and counts its closes.
type quietModel struct{ closed *int }

func (quietModel) Open(context.Context, []byte) (transcribe.Recording, error) { return nil, nil }

func (quietModel) Transcribe(context.Context, transcribe.Audio) (string, error) { return "", nil }

func (m quietModel) Close() error {
	*m.closed++
	return nil
}

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
		var got settings.Config
		var opens, closed int
		model := NewHolder(settings.Defaults(), func(_ context.Context, cfg settings.Config) (Model, error) {
			opens++
			got = cfg
			return quietModel{closed: &closed}, nil
		})
		model.UpdateSettings(tc.values)
		first, err := model.GetRecognizer(t.Context())
		if err != nil || first == nil {
			t.Fatalf("GetRecognizer() = %v, %v", first, err)
		}
		second, err := model.GetRecognizer(t.Context())
		if err != nil || second != first || opens != 1 {
			t.Fatalf("second GetRecognizer() = %v, %v (opens=%d)", second, err, opens)
		}
		if got.Model.Name != tc.model || !strings.Contains(got.Model.Repo, tc.repo) {
			t.Errorf("settings %v: got %s from %s, want %s from %s",
				tc.values, got.Model.Name, got.Model.Repo, tc.model, tc.repo)
		}
		if err := model.Close(); err != nil || closed != 1 {
			t.Errorf("Close() = %v with %d closed, want the open model closed once", err, closed)
		}
	}

	errOpen := errors.New("open failed")
	failModels := NewHolder(settings.Defaults(), func(context.Context, settings.Config) (Model, error) {
		return nil, errOpen
	})
	if _, err := failModels.GetRecognizer(t.Context()); !errors.Is(err, errOpen) {
		t.Fatalf("expected errOpen, got %v", err)
	}
}
