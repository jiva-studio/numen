package text_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestWebVTTRoundTripsCues(t *testing.T) {
	encoded := text.MarshalWebVTT([]domain.TranscriptCue{
		{Text: "first caption", From: 1500, To: 4200},
		{Text: "  ", From: 4200, To: 5000},
		{Text: "second caption", From: 3723456, To: 3725000},
	})

	words, cues := text.ParseWebVTT(encoded)

	if words != "first caption\nsecond caption" {
		t.Errorf("the captions read %q", words)
	}
	want := []domain.TranscriptCue{
		{Text: "first caption", From: 1500, To: 4200, Offset: 0},
		{Text: "second caption", From: 3723456, To: 3725000, Offset: 14},
	}
	if !slices.Equal(cues, want) {
		t.Errorf("the cues are %+v, want %+v", cues, want)
	}
}

func TestANoteANamedCueAndTimingsOtherToolsWriteAreRead(t *testing.T) {
	words, cues := text.ParseWebVTT([]byte("WEBVTT\r\n\r\nNOTE encoded elsewhere\r\n\r\ncue-1\r\n01:02.500 --> 00:01:03.000 line:0\r\nsaid\r\n"))

	if words != "said" || len(cues) != 1 || cues[0].From != 62500 || cues[0].To != 63000 {
		t.Errorf("read %q and %+v", words, cues)
	}
}
