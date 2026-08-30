package flashcards_test

import (
	"encoding/json"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

// A build that keeps nothing works the schedules out from the answers at every
// launch, and says what a build that keeps them says. The cache is a saving and
// never an answer of its own.
func TestSchedulesAreTheSameWithNothingKept(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}

	kept, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	bare := s.kept
	bare.Kept = nil
	worked, err := bare.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if len(worked) != len(kept) {
		t.Fatalf("kept %d schedules, worked out %d", len(kept), len(worked))
	}
	if !worked[on].Due.Equal(kept[on].Due) || worked[on].Reps != kept[on].Reps {
		t.Errorf("worked out %+v, kept %+v", worked[on], kept[on])
	}
}

// The cache is written in one order whatever order the schedules were worked
// out in, so two machines that answered the same cards hold the same file and a
// synchroniser has nothing to reconcile.
func TestTheCacheIsWrittenInOneOrder(t *testing.T) {
	s := opened(t, vault)
	// Both faces of one card, so the order turns on the face and not the card.
	recognise := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	name := history.CardFace{Card: "k7m2xq9fzp", Face: "Name it"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), name, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := record.Answer(t.Context(), recognise, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := s.kept.Execute(t.Context(), s.vault); err != nil {
		t.Fatal(err)
	}

	raw, err := s.kept.Kept.Read(t.Context(), s.vault.ID)
	if err != nil {
		t.Fatal(err)
	}
	var was plantedCache
	if err := json.Unmarshal(raw, &was); err != nil {
		t.Fatal(err)
	}

	if len(was.Faces) != 2 {
		t.Fatalf("the cache holds %d faces, want the two answered", len(was.Faces))
	}
	if was.Faces[0].Face != "Name it" || was.Faces[1].Face != "Recognise" {
		t.Errorf("the cache holds %s then %s", was.Faces[0].Face, was.Faces[1].Face)
	}
}

// A cache nothing can read is nothing remembered: the answers are there, and
// the schedules are worked out from them again.
func TestACacheNothingCanReadIsWorkedOutAgain(t *testing.T) {
	s := opened(t, vault)
	on := history.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := s.run(t, time.Now()).Answer(t.Context(), on, history.Good, 0); err != nil {
		t.Fatal(err)
	}
	want, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}

	if err := s.kept.Kept.Write(t.Context(), s.vault.ID, []byte("not a cache")); err != nil {
		t.Fatal(err)
	}

	got, err := s.kept.Execute(t.Context(), s.vault)
	if err != nil {
		t.Fatalf("a cache nothing can read refused the schedules: %v", err)
	}
	if !got[on].Due.Equal(want[on].Due) {
		t.Errorf("worked out %+v, want %+v", got[on], want[on])
	}
}
