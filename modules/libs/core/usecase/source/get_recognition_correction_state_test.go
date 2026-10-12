package source

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

func TestRecognitionCorrectionsFollowTheNamesTheProofreaderWrites(t *testing.T) {
	s := newTextScene(t)
	s.index.put(first.ID, domain.Source{
		Fingerprint: domain.Fingerprint{Path: scanPath, Kind: domain.KindBook},
		Producer:    text.OCR, Hash: "scanhash",
	})
	state := func() RunState {
		got, err := s.texts.GetRecognitionCorrectionState.Execute(t.Context(), s.vault, scanPath)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := state(); got != RunNone {
		t.Fatalf("state is %v before anything", got)
	}
	s.write(t, text.Corrections(text.OCR, "scanhash"), "begun")
	if got := state(); got != RunActive {
		t.Fatalf("state is %v with corrections begun", got)
	}
	s.write(t, text.Proofread(text.OCR, "scanhash"), "done")
	if got := state(); got != RunComplete {
		t.Fatalf("state is %v with the proofreading recorded", got)
	}
}
