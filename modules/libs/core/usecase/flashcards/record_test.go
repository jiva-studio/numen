package flashcards_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// An answer belongs to one card through one face. A line naming neither stands
// for nothing a schedule could be worked out from, so it is refused before it
// is written rather than left in the log for every later reading to skip.
func TestAnAnswerSaysWhichCardAndThroughWhichFace(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
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

// Taking back an answer names the one it takes back. Nothing here guesses which
// of a session's answers a person meant.
func TestTakingBackNamesTheAnswerItTakesBack(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)

	if _, err := s.run(t, time.Now()).TakeBack(t.Context(), ""); err == nil {
		t.Error("an answer taken back without naming one was written")
	}
}

// A run is known by the file it writes, which is what the counting is kept
// against: a run of the same name and length holds the same answers.
func TestARunIsKnownByTheFileItWrites(t *testing.T) {
	t.Parallel()
	s := opened(t, vault)
	on := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}

	record := s.run(t, time.Now())
	if _, err := record.Answer(t.Context(), on, review.Good, 0); err != nil {
		t.Fatal(err)
	}

	name := record.Run.Name()
	if !strings.HasSuffix(name, ".jsonl") {
		t.Errorf("the run is known as %q", name)
	}
	held := runsOf(t, s)
	if len(held) != 1 || !strings.HasSuffix(name, held[0]) {
		t.Errorf("the run is known as %q and the vault holds %v", name, held)
	}
}
