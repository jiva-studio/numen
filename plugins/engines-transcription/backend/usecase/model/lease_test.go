package model

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/speech"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/ports"
)

// closeErrors keeps the failures to close a model the holder reports.
type closeErrors struct{ errs []error }

func (c *closeErrors) ReportCloseError(err error) { c.errs = append(c.errs, err) }

// heldRecording is a recording that receives no audio.
type heldRecording struct{}

func (heldRecording) GetDuration() int { return 0 }

func (heldRecording) GetSegments(context.Context, int, int) ([]speech.Audio, error) { return nil, nil }

func (heldRecording) Close() error { return nil }

// usedModel counts the recordings opened on it and its closes, fails its close with closeErr, and
// answers each transcription with its name. When transcribing is set, a transcription signals on it
// and waits for released.
type usedModel struct {
	name         string
	opens        int
	closes       int
	closeErr     error
	transcribing chan struct{}
	released     chan struct{}
}

func (m *usedModel) Open(context.Context, []byte) (ports.Recording, error) {
	m.opens++
	return heldRecording{}, nil
}

func (m *usedModel) Transcribe(context.Context, speech.Audio) (string, error) {
	if m.transcribing != nil {
		m.transcribing <- struct{}{}
		<-m.released
	}
	return m.name, nil
}

func (m *usedModel) Close() error {
	m.closes++
	return m.closeErr
}

// usedModels opens a usedModel for each request and keeps them in the order they opened.
type usedModels struct {
	models   []*usedModel
	closeErr error
	isHeld   bool
}

func (u *usedModels) open(_ context.Context, cfg settings.Config) (Model, error) {
	m := &usedModel{name: cfg.Model.Name, opens: 0, closes: 0, closeErr: u.closeErr, transcribing: nil, released: nil}
	if u.isHeld {
		m.transcribing = make(chan struct{})
		m.released = make(chan struct{})
	}
	u.models = append(u.models, m)
	return m, nil
}

func TestAModelReplacedWhileARecordingIsOpenClosesOnceWhenTheRecordingCloses(t *testing.T) {
	opened := &usedModels{}
	models := NewHolder(settings.NewDefaultConfig(), opened.open, &closeErrors{})
	recognizer, err := models.GetRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	recording, err := recognizer.Open(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	if got := opened.models[0].closes; got != 0 {
		t.Fatalf("the parakeet-tdt-0.6b-v3-int8 model was closed %d times while a recording was open, want 0", got)
	}
	if err := recording.Close(); err != nil {
		t.Fatal(err)
	}
	if err := recording.Close(); err != nil {
		t.Fatal(err)
	}
	if err := models.Close(); err != nil {
		t.Fatal(err)
	}

	if got := opened.models[0].closes; got != 1 {
		t.Errorf("the parakeet-tdt-0.6b-v3-int8 model was closed %d times after its recording closed, want 1", got)
	}
}

func TestAModelReplacedWhileATranscriptionRunsClosesWhenTheTranscriptionReturns(t *testing.T) {
	opened := &usedModels{isHeld: true}
	models := NewHolder(settings.NewDefaultConfig(), opened.open, &closeErrors{})
	recognizer, err := models.GetRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	old := opened.models[0]
	type answer struct {
		words string
		err   error
	}
	answered := make(chan answer)
	go func() {
		words, err := recognizer.Transcribe(t.Context(), speech.Audio{})
		answered <- answer{words: words, err: err}
	}()
	<-old.transcribing

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	if old.closes != 0 {
		t.Fatalf("the parakeet-tdt-0.6b-v3-int8 model was closed %d times while a transcription ran, want 0", old.closes)
	}
	close(old.released)
	got := <-answered

	if got.err != nil || got.words != "parakeet-tdt-0.6b-v3-int8" {
		t.Errorf("Transcribe() = %q, %v, want the transcription finished on parakeet-tdt-0.6b-v3-int8", got.words, got.err)
	}
	if old.closes != 1 {
		t.Errorf("the parakeet-tdt-0.6b-v3-int8 model was closed %d times after the transcription returned, want 1", old.closes)
	}
}

func TestAFailedCloseIsReportedOnceAndTheNextRequestOpensTheNewModel(t *testing.T) {
	for _, tc := range []struct {
		name        string
		isRecording bool
	}{
		{"no recording open", false},
		{"a recording open", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errClose := errors.New("close failed")
			opened := &usedModels{closeErr: errClose}
			reported := &closeErrors{}
			models := NewHolder(settings.NewDefaultConfig(), opened.open, reported)
			recognizer, err := models.GetRecognizer(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			var recording ports.Recording
			if tc.isRecording {
				if recording, err = recognizer.Open(t.Context(), nil); err != nil {
					t.Fatal(err)
				}
			}

			models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
			if recording != nil {
				if err := recording.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if len(reported.errs) != 1 || !errors.Is(reported.errs[0], errClose) {
				t.Fatalf("reported %v, want the close failure once", reported.errs)
			}
			next, err := models.GetRecognizer(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			words, err := next.Transcribe(t.Context(), speech.Audio{})

			if err != nil || words != "parakeet-tdt-0.6b-v2-int8" || len(opened.models) != 2 {
				t.Errorf("next Transcribe() = %q, %v with %d models opened, want parakeet-tdt-0.6b-v2-int8 opened second", words, err, len(opened.models))
			}
			if len(reported.errs) != 1 {
				t.Errorf("reported %v, want the close failure once", reported.errs)
			}
		})
	}
}

func TestARecordingRequestedFromAReplacedModelOpensOnTheNewModel(t *testing.T) {
	for _, tc := range []struct {
		name        string
		isRecording bool
		oldOpens    int
	}{
		{"the replaced model closed", false, 0},
		{"the replaced model waiting for its recording", true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opened := &usedModels{}
			models := NewHolder(settings.NewDefaultConfig(), opened.open, &closeErrors{})
			old, err := models.GetRecognizer(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if tc.isRecording {
				if _, err := old.Open(t.Context(), nil); err != nil {
					t.Fatal(err)
				}
			}
			models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})

			recording, err := old.Open(t.Context(), nil)

			if err != nil || recording == nil {
				t.Fatalf("Open() on the replaced model = %v, %v", recording, err)
			}
			if len(opened.models) != 2 || opened.models[1].name != "parakeet-tdt-0.6b-v2-int8" || opened.models[1].opens != 1 {
				t.Fatalf("opened %d models, want the recording on parakeet-tdt-0.6b-v2-int8", len(opened.models))
			}
			if got := opened.models[0].opens; got != tc.oldOpens {
				t.Errorf("parakeet-tdt-0.6b-v3-int8 opened %d recordings, want only the one before the replacement", got)
			}
			if err := recording.Close(); err != nil {
				t.Fatal(err)
			}
			if got := opened.models[1].closes; got != 0 {
				t.Errorf("parakeet-tdt-0.6b-v2-int8 was closed %d times when its recording closed, want 0", got)
			}
		})
	}
}

func TestClosingTheHolderClosesTheOpenModelAndTheReplacedOneWhenItsLastRecordingCloses(t *testing.T) {
	opened := &usedModels{}
	models := NewHolder(settings.NewDefaultConfig(), opened.open, &closeErrors{})
	old, err := models.GetRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	recording, err := old.Open(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})
	if _, err := models.GetRecognizer(t.Context()); err != nil {
		t.Fatal(err)
	}

	if err := models.Close(); err != nil {
		t.Fatal(err)
	}
	if opened.models[1].closes != 1 || opened.models[0].closes != 0 {
		t.Fatalf("after Close parakeet-tdt-0.6b-v2-int8 closed %d times and parakeet-tdt-0.6b-v3-int8 %d times, want 1 and 0",
			opened.models[1].closes, opened.models[0].closes)
	}
	if err := recording.Close(); err != nil {
		t.Fatal(err)
	}

	if got := opened.models[0].closes; got != 1 {
		t.Errorf("parakeet-tdt-0.6b-v3-int8 was closed %d times after its recording closed, want 1", got)
	}
}

func TestARecordingClosedTwiceEndsItsUseOfTheModelOnce(t *testing.T) {
	opened := &usedModels{}
	models := NewHolder(settings.NewDefaultConfig(), opened.open, &closeErrors{})
	recognizer, err := models.GetRecognizer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	first, err := recognizer.Open(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recognizer.Open(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	models.UpdateSettings(map[string]string{"model": "parakeet-tdt-0.6b-v2-int8"})

	if got := opened.models[0].closes; got != 0 {
		t.Errorf("parakeet-tdt-0.6b-v3-int8 was closed %d times while its second recording was open, want 0", got)
	}
}
