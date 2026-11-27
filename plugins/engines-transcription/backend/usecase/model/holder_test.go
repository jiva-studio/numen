package model

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/speech"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

// quietModel transcribes nothing and counts its closes.
type quietModel struct{ closed *int }

func (quietModel) Open(context.Context, []byte) (ports.Recording, error) { return nil, nil }

func (quietModel) Transcribe(context.Context, speech.Audio) (string, error) { return "", nil }

func (m quietModel) Close() error {
	*m.closed++
	return nil
}

func TestTheRecognizerOpensWithTheModelOfTheSettings(t *testing.T) {
	for _, tc := range []struct {
		values  map[string]string
		model   string
		encoder string
	}{
		{map[string]string{}, "parakeet-tdt-0.6b-v3-int8", "parakeet-v3-encoder"},
		{map[string]string{"model": "parakeet-tdt-0.6b-v3-int8"}, "parakeet-tdt-0.6b-v3-int8", "parakeet-v3-encoder"},
		{map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"}, "parakeet-tdt-0.6b-v2-int8", "parakeet-v2-encoder"},
		{map[string]string{"model": "whisper-large"}, "parakeet-tdt-0.6b-v3-int8", "parakeet-v3-encoder"},
	} {
		var got settings.Config
		var opens, closed int
		model := NewHolder(settings.NewDefaultConfig(), func(_ context.Context, cfg settings.Config) (Model, error) {
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
		if got.Model.Name != tc.model || got.Model.Dependencies.Encoder != tc.encoder {
			t.Errorf("settings %v: got %s with %s, want %s with %s",
				tc.values, got.Model.Name, got.Model.Dependencies.Encoder, tc.model, tc.encoder)
		}
		if err := model.Close(); err != nil || closed != 1 {
			t.Errorf("Close() = %v with %d closed, want the open model closed once", err, closed)
		}
	}

	errOpen := errors.New("open failed")
	failModels := NewHolder(settings.NewDefaultConfig(), func(context.Context, settings.Config) (Model, error) {
		return nil, errOpen
	})
	if _, err := failModels.GetRecognizer(t.Context()); !errors.Is(err, errOpen) {
		t.Fatalf("expected errOpen, got %v", err)
	}
	if err := failModels.Close(); err != nil {
		t.Fatalf("expected Close on unopened holder to return nil, got %v", err)
	}
}

func TestTheModelIsNotOpenedBeforeTheFirstRecognizerRequest(t *testing.T) {
	var opens int
	models := NewHolder(settings.NewDefaultConfig(), func(context.Context, settings.Config) (Model, error) {
		opens++
		return quietModel{closed: new(int)}, nil
	})

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})

	if opens != 0 {
		t.Errorf("the model was opened %d times before a request", opens)
	}
}

func TestAnOpenThatFailedForMissingFilesIsTriedAgainAtTheNextRequest(t *testing.T) {
	filesArrived := false
	models := NewHolder(settings.NewDefaultConfig(), func(context.Context, settings.Config) (Model, error) {
		if !filesArrived {
			return nil, errors.New("model encoder.onnx not found")
		}
		return quietModel{closed: new(int)}, nil
	})
	if _, err := models.GetRecognizer(t.Context()); err == nil {
		t.Fatal("the open succeeded without the files")
	}

	filesArrived = true
	recognizer, err := models.GetRecognizer(t.Context())

	if err != nil || recognizer == nil {
		t.Errorf("GetRecognizer() after the files arrived = %v, %v", recognizer, err)
	}
}

// openedModels opens a quietModel for each request and keeps the name each opened with and the closes of each.
type openedModels struct {
	names  []string
	closed []*int
}

func (o *openedModels) open(_ context.Context, cfg settings.Config) (Model, error) {
	o.names = append(o.names, cfg.Model.Name)
	o.closed = append(o.closed, new(int))
	return quietModel{closed: o.closed[len(o.closed)-1]}, nil
}

func TestAnotherModelInTheSettingsClosesTheOpenModelAndTheNextRequestOpensTheNewOne(t *testing.T) {
	opened := &openedModels{}
	models := NewHolder(settings.NewDefaultConfig(), opened.open)
	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v3-int8"})
	if _, err := models.GetRecognizer(t.Context()); err != nil {
		t.Fatal(err)
	}

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	if *opened.closed[0] != 1 {
		t.Errorf("the parakeet-tdt-0.6b-v3-int8 model was closed %d times after the settings named another, want 1", *opened.closed[0])
	}
	if _, err := models.GetRecognizer(t.Context()); err != nil {
		t.Fatal(err)
	}

	if len(opened.names) != 2 || opened.names[1] != "parakeet-tdt-0.6b-v2-int8" {
		t.Errorf("opened %v, want parakeet-tdt-0.6b-v3-int8 then parakeet-tdt-0.6b-v2-int8", opened.names)
	}
}

func TestTheSameModelInTheSettingsKeepsTheOpenModel(t *testing.T) {
	opened := &openedModels{}
	models := NewHolder(settings.NewDefaultConfig(), opened.open)
	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	first, err := models.GetRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	second, err := models.GetRecognizer(t.Context())

	if err != nil || second != first || len(opened.names) != 1 || *opened.closed[0] != 0 {
		t.Errorf("after the same model: %v, %v, opened %v, closed %d; want the open model kept", second, err, opened.names, *opened.closed[0])
	}
}
