package source

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestTranscriptWriteKeepsWhatWasHeard(t *testing.T) {
	s := newTextScene(t)
	s.write(t, domain.NewArtifactName(domain.ArtifactText, domain.ASR, talkHash), "heard")
	cues := []domain.TranscriptCue{{Text: "put right", From: 0, To: 500}}
	var cutAt string

	_, err := s.texts.WriteTranscript.Execute(t.Context(), s.vault, recordingPath, cues,
		func(_ context.Context, _ domain.Vault, path string) error { cutAt = path; return nil })
	if err != nil {
		t.Fatal(err)
	}

	heard, _ := s.shelf.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactText, domain.ASR, talkHash))
	if string(heard) != "heard" {
		t.Errorf("what was heard became %q", heard)
	}
	written, _ := s.shelf.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, talkHash))
	want := append(transcript.Marshal(cues), transcript.GetHandWrittenMark()...)
	if string(written) != string(want) {
		t.Errorf("corrections are %q, want %q", written, want)
	}
	if cutAt != recordingPath {
		t.Errorf("the recording was cut again at %q", cutAt)
	}
}

func TestTranscriptWriteSurvivesACutFailure(t *testing.T) {
	s := newTextScene(t)
	failure := errors.New("no cutter")

	written, err := s.texts.WriteTranscript.Execute(t.Context(), s.vault, recordingPath,
		[]domain.TranscriptCue{{Text: "right", From: 0, To: 1}},
		func(context.Context, domain.Vault, string) error { return failure })
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(written.CutError, failure) {
		t.Errorf("cut error is %v, want the cutter's", written.CutError)
	}
	if _, err := s.shelf.ReadArtifact(t.Context(), domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, talkHash)); err != nil {
		t.Errorf("the corrections did not land: %v", err)
	}
}

func TestTranscriptWriteIsRefusedWhileARunHoldsIt(t *testing.T) {
	s := newTextScene(t)
	s.claim(t, domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, talkHash))

	_, err := s.texts.WriteTranscript.Execute(t.Context(), s.vault, recordingPath, nil, nil)
	if !errors.Is(err, ErrTranscriptBusy) {
		t.Errorf("error is %v, want ErrTranscriptBusy", err)
	}
}

func TestTranscriptWriteIsRefusedForARecordingNothingHeard(t *testing.T) {
	s := newTextScene(t).withoutIndex()

	_, err := s.texts.WriteTranscript.Execute(t.Context(), s.vault, recordingPath, nil, nil)
	if !errors.Is(err, ErrNoTranscript) {
		t.Errorf("error is %v, want ErrNoTranscript", err)
	}
}
