package flashcards_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/flashcards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// An answer belongs to one card through one face. A line naming neither stands
// for nothing a schedule could be worked out from, so it is refused before it
// is written and not left in the log for every later reading to skip.
func TestAnAnswerSaysWhichCardAndThroughWhichFace(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	record := s.run(t, time.Now())

	for _, on := range []review.CardFaceID{
		{Card: "", Face: "Recognise"},
		{Card: "k7m2xq9fzp", Face: ""},
		{},
	} {
		if _, err := record.Answer(t.Context(), on, review.Good, 0); err == nil {
			t.Errorf("an answer on %+v was written", on)
		}
	}

	held, err := flashcards.Log{Stores: s.logs}.Read(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(held.Answers) != 0 {
		t.Errorf("the vault holds %d answers, want none", len(held.Answers))
	}
}

// A Record built with no clock refuses an answer and does not panic.
func TestARecordWithNoClockRefusesAnAnswer(t *testing.T) {
	t.Parallel()
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	if _, err := (flashcards.Record{}).Answer(t.Context(), nil, on, review.Good, 0); err == nil {
		t.Error("an answer stamped by no clock was written")
	}
	if _, err := (flashcards.Record{}).TakeBack(t.Context(), nil, "a"); err == nil {
		t.Error("a taking back stamped by no clock was written")
	}
}

// Taking back an answer names the one it takes back. Nothing here guesses which
// of a session's answers a person meant.
func TestTakingBackNamesTheAnswerItTakesBack(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)

	if _, err := s.run(t, time.Now()).TakeBack(t.Context(), ""); !errors.Is(err, flashcards.ErrNoAnswer) {
		t.Errorf("an answer taken back without naming one came back with %v", err)
	}
}

// A run is known by the file it writes, which is what the counting is kept
// against: a run of the same name and length holds the same answers.
func TestARunIsKnownByTheFileItWrites(t *testing.T) {
	t.Parallel()
	s := openVault(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	now := time.Now()
	run, err := flashcards.Log{Stores: s.logs}.Open(t.Context(), s.vault, now)
	if err != nil {
		t.Fatal(err)
	}
	record := newSitting(run, func() time.Time { return now })
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	name := run.GetName()
	held := getRunFiles(t, s)
	if len(held) != 1 || held[0] != name+".jsonl" {
		t.Errorf("the run is known as %q and the vault holds %v", name, held)
	}
}
