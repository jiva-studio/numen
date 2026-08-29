package flashcards_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards"
)

func answered(id, card, face, when string, r flashcards.Rating) flashcards.Answer {
	return flashcards.Answer{
		ID:       id,
		CardFace: flashcards.CardFace{Card: card, Face: face},
		At:       at(when),
		Rating:   r,
	}
}

// A file synchronised from another machine lands after answers later than the
// ones in it. The order a schedule is worked out in is the order the answers
// were given, and never the order the files were read.
func TestAnswersAreCountedInTheOrderTheyWereGiven(t *testing.T) {
	first := answered("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", flashcards.Again)
	second := answered("01B", "k7m2xq9fzp", "Recognise", "2026-08-21T09:00:00Z", flashcards.Good)
	third := answered("01C", "k7m2xq9fzp", "Recognise", "2026-08-25T09:00:00Z", flashcards.Good)

	by := flashcards.NewFSRS()
	want := flashcards.Replay(by, []flashcards.Answer{first, second, third})
	got := flashcards.Replay(by, []flashcards.Answer{third, first, second})

	shown := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if got[shown] != want[shown] {
		t.Errorf("read out of order gave %+v, want %+v", got[shown], want[shown])
	}
}

// Two answers of one instant are counted in the order their identifiers were
// minted in, so a replay of one history is one schedule however it is read.
func TestTwoAnswersOfOneInstantKeepTheirOrder(t *testing.T) {
	early := answered("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", flashcards.Again)
	late := answered("01B", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", flashcards.Easy)

	by := flashcards.NewFSRS()
	shown := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	one := flashcards.Replay(by, []flashcards.Answer{early, late})[shown]
	other := flashcards.Replay(by, []flashcards.Answer{late, early})[shown]
	if one != other {
		t.Errorf("one history read two ways gave %+v and %+v", one, other)
	}
}

// A person who took an answer back is not counted as having given it, and both
// lines stay in the file.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	given := answered("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", flashcards.Again)
	back := flashcards.Answer{ID: "01B", At: at("2026-08-20T09:00:04Z"), Undoes: "01A"}

	left := flashcards.Replay(flashcards.NewFSRS(), []flashcards.Answer{given, back})
	if len(left) != 0 {
		t.Errorf("a card face whose only answer was taken back has no schedule, got %+v", left)
	}
}

// Each face of a stencil asks a different thing, so each has a path of its own.
func TestEachFaceOfACardIsScheduledOnItsOwn(t *testing.T) {
	recognise := answered("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", flashcards.Easy)
	name := answered("01B", "k7m2xq9fzp", "Name it", "2026-08-20T09:00:10Z", flashcards.Again)

	left := flashcards.Replay(flashcards.NewFSRS(), []flashcards.Answer{recognise, name})
	if len(left) != 2 {
		t.Fatalf("two faces answered left %d schedules", len(left))
	}
	easy := left[flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}]
	again := left[flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Name it"}]
	if !easy.Due.After(again.Due) {
		t.Errorf("the face that came back easily is due %v, the one that did not %v", easy.Due, again.Due)
	}
}

// A schedule is worked out again whenever it is wanted, so working it out twice
// gives the same answer.
func TestReplayingOneHistoryTwiceGivesOneSchedule(t *testing.T) {
	var history []flashcards.Answer
	when := at("2026-01-01T09:00:00Z")
	for i, r := range []flashcards.Rating{flashcards.Good, flashcards.Again, flashcards.Hard, flashcards.Good, flashcards.Easy} {
		history = append(history, flashcards.Answer{
			ID:       string(rune('A'+i)) + "01",
			CardFace: flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"},
			At:       when.Add(time.Duration(i) * 24 * time.Hour),
			Rating:   r,
		})
	}

	by := flashcards.NewFSRS()
	shown := flashcards.CardFace{Card: "k7m2xq9fzp", Face: "Recognise"}
	if one, other := flashcards.Replay(by, history)[shown], flashcards.Replay(by, history)[shown]; one != other {
		t.Errorf("one history gave %+v and then %+v", one, other)
	}
}

// A card face nobody has answered has no schedule, and that is what a person
// means by a new card.
func TestACardFaceNobodyAnsweredHasNoSchedule(t *testing.T) {
	left := flashcards.Replay(flashcards.NewFSRS(), nil)
	if len(left) != 0 {
		t.Errorf("no answers left %d schedules", len(left))
	}
	if (flashcards.Schedule{}).Seen() {
		t.Error("a schedule nobody has answered says it has been seen")
	}
}
