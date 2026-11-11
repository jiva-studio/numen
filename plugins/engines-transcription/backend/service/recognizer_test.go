package service

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/model"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

func TestTheRecognizerTranscribesThroughTheEngine(t *testing.T) {
	holder := model.NewHolder(settings.Defaults(), newOpener(engine.NewRuntime(), true))

	recognizer, err := holder.GetRecognizer(t.Context())
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
	if err := holder.Close(); err != nil {
		t.Error(err)
	}
}
