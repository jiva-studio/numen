package flashcards_test

import (
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// A run is listed and then taken away by another machine's synchroniser before
// it can be read. That is a file gone, not a vault whose history cannot be
// read: everything else the person answered is still theirs.
func TestARunTakenAwayBeforeItWasReadIsGone(t *testing.T) {
	s := opened(t, vault)
	store, err := s.logs.Open(s.vault)
	if err != nil {
		t.Fatal(err)
	}

	ran, err := flashcards.Log{Stores: s.logs}.Run(t.Context(), store, port.Stored{
		Name: "flashcards/01ARZ3NDEKTSV4RRFFQ69G5FAV.jsonl",
	})
	if err != nil {
		t.Fatalf("a run that is no longer there was trouble: %v", err)
	}
	if !ran.Gone {
		t.Error("a run that is no longer there did not say so")
	}
	if len(ran.Answers) != 0 {
		t.Errorf("a run that is no longer there held %d answers", len(ran.Answers))
	}
}

// What a vault holds is every answer of every run, and the runs it was read
// from at the length they were read at — which is what a cache is measured
// against.
func TestWhatAVaultHoldsIsEveryRunItWasReadFrom(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	other := history.CardFace{Card: "zpqrstvwxy", Face: "Recognise"}

	if _, err := s.run(t, time.Now().AddDate(0, 0, -1)).Answer(
		t.Context(), on, history.Good, 0,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run(t, time.Now()).Answer(t.Context(), other, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 2 {
		t.Errorf("the vault holds %d answers, want the two written", len(held.Answers))
	}
	if len(held.Files) != 2 {
		t.Errorf("read from %d runs, want the two sittings", len(held.Files))
	}
	for _, one := range held.Files {
		if one.Size == 0 {
			t.Errorf("%s was read at no length at all", one.Name)
		}
	}
	if held.Skipped != 0 {
		t.Errorf("%d lines could not be acted on, want none", held.Skipped)
	}
}

// A vault nobody has reviewed holds no folder and no files, which is an answer
// and not a failure.
func TestAVaultNobodyReviewedHoldsNoRuns(t *testing.T) {
	s := opened(t, vault)

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 0 || len(held.Files) != 0 {
		t.Errorf("read %+v", held)
	}
}
