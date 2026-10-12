package source

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

func TestTranscriptCorrectionsAreCompleteWhenWrittenAndActiveWhileARunHoldsTheName(t *testing.T) {
	s := newTextScene(t)
	state := func() RunState {
		got, err := s.texts.GetTranscriptCorrectionState.Execute(t.Context(), s.vault, recordingPath)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := state(); got != RunNone {
		t.Fatalf("state is %v before anything", got)
	}
	s.claim(t, text.Partial(text.ASR, talkHash))
	if got := state(); got != RunActive {
		t.Fatalf("state is %v while a run holds the recording", got)
	}
	s.write(t, text.Corrections(text.ASR, talkHash), "right")
	if got := state(); got != RunComplete {
		t.Fatalf("state is %v with corrections written", got)
	}
}
