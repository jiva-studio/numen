package source

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestTranscriptIsWhatAPersonInsertRightBeforeWhatWasHeard(t *testing.T) {
	s := newTextScene(t)
	read := func() string {
		got, err := s.texts.GetTranscript.Execute(t.Context(), s.vault, recordingPath)
		if err != nil {
			t.Fatal(err)
		}
		return string(got.Words)
	}
	readRaw := func() string {
		got, err := s.texts.GetRawTranscript.Execute(t.Context(), s.vault, recordingPath)
		if err != nil {
			t.Fatal(err)
		}
		return string(got.Words)
	}
	if got := read(); got != "" {
		t.Fatalf("nothing is heard yet, got %q", got)
	}

	s.write(t, domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, talkHash), "partial")
	if got := read(); got != "partial" {
		t.Errorf("a run still going is read as far as it got, got %q", got)
	}
	s.write(t, domain.NewArtifactName(domain.ArtifactText, domain.ASR, talkHash), "heard")
	s.write(t, domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, talkHash), "right")

	if got := read(); got != "right" {
		t.Errorf("the words as they stand are the corrections, got %q", got)
	}
	if got := readRaw(); got != "heard" {
		t.Errorf("the raw words leave corrections out, got %q", got)
	}
}

func TestATranscriptOfARecordingNothingHeardIsNotFound(t *testing.T) {
	s := newTextScene(t).withoutIndex()

	got, err := s.texts.GetTranscript.Execute(t.Context(), s.vault, recordingPath)
	if err != nil || got.Exists {
		t.Errorf("found %v, %v", got.Exists, err)
	}
}
