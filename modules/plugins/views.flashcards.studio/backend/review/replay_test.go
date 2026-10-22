package review_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

func makeAnswer(id, card, face, when string, r review.Rating) review.Answer {
	return review.Answer{
		ID:       id,
		CardFace: review.CardFaceID{Card: card, Face: face},
		At:       parseTime(when),
		Rating:   r,
	}
}

// A file synchronised from another machine lands after answers later than the
// ones in it. The order a schedule is worked out in is the order the answers
// were given, and never the order the files were read.
func TestAnswersAreCountedInTheOrderTheyWereGiven(t *testing.T) {
	first := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	second := makeAnswer("01B", "k7m2xq9fzp", "Recognise", "2026-08-21T09:00:00Z", review.Good)
	third := makeAnswer("01C", "k7m2xq9fzp", "Recognise", "2026-08-25T09:00:00Z", review.Good)

	by := review.NewFSRS()
	want := review.OrderAnswers([]review.Answer{first, second, third}).Replay(ahead, review.ScheduleBy(by))
	got := review.OrderAnswers([]review.Answer{third, first, second}).Replay(ahead, review.ScheduleBy(by))

	cardFace := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if got[cardFace] != want[cardFace] {
		t.Errorf("read out of order gave %+v, want %+v", got[cardFace], want[cardFace])
	}
}

// Two answers of one instant are counted in the order their identifiers were
// minted in, so a replay of one history is one schedule however it is read.
func TestTwoAnswersOfOneInstantKeepTheirOrder(t *testing.T) {
	early := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	late := makeAnswer("01B", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Easy)

	by := review.NewFSRS()
	cardFace := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	one := review.OrderAnswers([]review.Answer{early, late}).Replay(ahead, review.ScheduleBy(by))[cardFace]
	other := review.OrderAnswers([]review.Answer{late, early}).Replay(ahead, review.ScheduleBy(by))[cardFace]
	if one != other {
		t.Errorf("one history read two ways gave %+v and %+v", one, other)
	}
}

// A person who took an answer back is not counted as having given it, and both
// lines stay in the file.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	provided := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	back := review.Answer{ID: "01B", At: parseTime("2026-08-20T09:00:04Z"), Undoes: "01A"}

	remaining := review.OrderAnswers([]review.Answer{provided, back}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 0 {
		t.Errorf("a card face whose only answer was taken back has no schedule, got %+v", remaining)
	}
}

// Each face of a stencil asks a different thing, so each has a path of its own.
func TestEachFaceOfACardIsScheduledOnItsOwn(t *testing.T) {
	recognise := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Easy)
	name := makeAnswer("01B", "k7m2xq9fzp", "Name it", "2026-08-20T09:00:10Z", review.Again)

	remaining := review.OrderAnswers([]review.Answer{recognise, name}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 2 {
		t.Fatalf("two faces answered left %d schedules", len(remaining))
	}
	easy := remaining[review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}]
	again := remaining[review.CardFaceID{Card: "k7m2xq9fzp", Face: "Name it"}]
	if !easy.Due.After(again.Due) {
		t.Errorf("the face that came back easily is due %v, the one that did not %v", easy.Due, again.Due)
	}
}

// A schedule is worked out again whenever it is wanted, so working it out twice
// gives the same answer.
func TestReplayingOneHistoryTwiceProducesOneSchedule(t *testing.T) {
	var history []review.Answer
	when := parseTime("2026-01-01T09:00:00Z")
	for i, r := range []review.Rating{review.Good, review.Again, review.Hard, review.Good, review.Easy} {
		history = append(history, review.Answer{
			ID:       string(rune('A'+i)) + "01",
			CardFace: review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"},
			At:       when.Add(time.Duration(i) * 24 * time.Hour),
			Rating:   r,
		})
	}

	by := review.NewFSRS()
	cardFace := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	if one, other := review.OrderAnswers(history).Replay(ahead, review.ScheduleBy(by))[cardFace], review.OrderAnswers(history).Replay(ahead, review.ScheduleBy(by))[cardFace]; one != other {
		t.Errorf("one history gave %+v and then %+v", one, other)
	}
}

// One identifier is one answer. A synchroniser that met a conflict leaves a
// second copy of a run beside the first, and every line in it is a line already
// counted: a card that came through this twice would be sent away for longer
// than it was earned.
func TestALineThatExistsTwiceIsCountedOnce(t *testing.T) {
	cardFace := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	one := makeAnswer("01A", cardFace.Card, cardFace.Face, "2026-08-20T09:00:00Z", review.Good)
	two := makeAnswer("01B", cardFace.Card, cardFace.Face, "2026-08-21T09:00:00Z", review.Good)

	by := review.NewFSRS()
	once := review.OrderAnswers([]review.Answer{one, two}).Replay(ahead, review.ScheduleBy(by))
	// The whole run again, as a conflicted copy of the file lands beside it.
	twice := review.OrderAnswers([]review.Answer{one, two, one, two}).Replay(ahead, review.ScheduleBy(by))

	if once[cardFace] != twice[cardFace] {
		t.Errorf("the same run twice left the card at %+v, once leaves it at %+v",
			twice[cardFace], once[cardFace])
	}
	if twice[cardFace].Reps != 2 {
		t.Errorf("the card was answered twice and has been through %d", twice[cardFace].Reps)
	}
}

// An answer taken back stays taken back however many copies of the line that
// took it back arrive.
func TestALineTakingAnAnswerBackTwiceConsumesItBackOnce(t *testing.T) {
	cardFace := review.CardFaceID{Card: "k7m2xq9fzp", Face: "Recognise"}
	provided := makeAnswer("01A", cardFace.Card, cardFace.Face, "2026-08-20T09:00:00Z", review.Good)
	back := review.Answer{ID: "01B", At: parseTime("2026-08-20T09:01:00Z"), Undoes: provided.ID}

	remaining := review.OrderAnswers([]review.Answer{provided, back, provided, back}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if _, target := remaining[cardFace]; target {
		t.Errorf("the answer was taken back and the card stands at %+v", remaining[cardFace])
	}
}

// means by a new card.
func TestACardFaceNobodyAnsweredHasNoSchedule(t *testing.T) {
	remaining := review.OrderAnswers(nil).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 0 {
		t.Errorf("no answers left %d schedules", len(remaining))
	}
	if !(review.Schedule{}).IsNew() {
		t.Error("a schedule nobody has answered says it is not new")
	}
}
