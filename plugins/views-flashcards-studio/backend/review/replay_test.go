package review_test

import (
	"fmt"
	"maps"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

func makeAnswer(id, card, reviewCard, when string, r review.Rating) review.Answer {
	return review.Answer{
		ID:         id,
		ReviewCard: review.ReviewCardID{Card: card, Template: reviewCard},
		At:         parseTime(when),
		Rating:     r,
	}
}

// A file synchronised from another machine lands after answers later than the
// ones in it. The order a schedule is worked out in is the order the answers
// were given, and never the order the files were read.
func TestAnswersAreCountedInTheOrderTheyWereGiven(t *testing.T) {
	first := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	second := makeAnswer("01B", "k7m2xq9fzp", "Recognise", "2026-08-21T09:00:00Z", review.Good)
	third := makeAnswer("01C", "k7m2xq9fzp", "Recognise", "2026-08-25T09:00:00Z", review.Good)

	delta := review.NewFSRS()
	want := review.OrderAnswers([]review.Answer{first, second, third}).Replay(ahead, review.ScheduleBy(delta))
	got := review.OrderAnswers([]review.Answer{third, first, second}).Replay(ahead, review.ScheduleBy(delta))

	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if got[reviewCard] != want[reviewCard] {
		t.Errorf("read out of order gave %+v, want %+v", got[reviewCard], want[reviewCard])
	}
}

// Two answers of one instant are counted in the order their identifiers were
// minted in, so a replay of one history is one schedule however it is read.
func TestTwoAnswersOfOneInstantKeepTheirOrder(t *testing.T) {
	early := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	late := makeAnswer("01B", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Easy)

	delta := review.NewFSRS()
	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	one := review.OrderAnswers([]review.Answer{early, late}).Replay(ahead, review.ScheduleBy(delta))[reviewCard]
	other := review.OrderAnswers([]review.Answer{late, early}).Replay(ahead, review.ScheduleBy(delta))[reviewCard]
	if one != other {
		t.Errorf("one history read two ways gave %+v and %+v", one, other)
	}
}

// An answer that was taken back is not counted as given, and both lines remain in
// the file.
func TestAnAnswerTakenBackIsNotCounted(t *testing.T) {
	provided := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Again)
	back := review.Answer{ID: "01B", At: parseTime("2026-08-20T09:00:04Z"), Undoes: "01A"}

	remaining := review.OrderAnswers([]review.Answer{provided, back}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 0 {
		t.Errorf("a review card whose only answer was taken back has no schedule, got %+v", remaining)
	}
}

// Each review card of a card model tests a different thing, so each has its own path.
func TestEachCardOfACardIsScheduledOnItsOwn(t *testing.T) {
	recognise := makeAnswer("01A", "k7m2xq9fzp", "Recognise", "2026-08-20T09:00:00Z", review.Easy)
	name := makeAnswer("01B", "k7m2xq9fzp", "Name it", "2026-08-20T09:00:10Z", review.Again)

	remaining := review.OrderAnswers([]review.Answer{recognise, name}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 2 {
		t.Fatalf("two review cards answered left %d schedules", len(remaining))
	}
	easy := remaining[review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}]
	again := remaining[review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Name it"}]
	if !easy.Due.After(again.Due) {
		t.Errorf("the review card that came back easily is due %v, the one that did not %v", easy.Due, again.Due)
	}
}

// A schedule is worked out again whenever it is wanted, so working it out twice
// gives the same answer.
func TestReplayingOneHistoryTwiceProducesOneSchedule(t *testing.T) {
	var history []review.Answer
	when := parseTime("2026-01-01T09:00:00Z")
	for i, r := range []review.Rating{review.Good, review.Again, review.Hard, review.Good, review.Easy} {
		history = append(history, review.Answer{
			ID:         string(rune('A'+i)) + "01",
			ReviewCard: review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"},
			At:         when.Add(time.Duration(i) * 24 * time.Hour),
			Rating:     r,
		})
	}

	delta := review.NewFSRS()
	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	if one, other := review.OrderAnswers(history).Replay(ahead, review.ScheduleBy(delta))[reviewCard], review.OrderAnswers(history).Replay(ahead, review.ScheduleBy(delta))[reviewCard]; one != other {
		t.Errorf("one history gave %+v and then %+v", one, other)
	}
}

// One identifier is one answer. A synchroniser that met a conflict leaves a
// second copy of a run beside the first, and every line in it is a line already
// counted: a card that came through this twice would be sent away for longer
// than it was earned.
func TestALineThatExistsTwiceIsCountedOnce(t *testing.T) {
	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	one := makeAnswer("01A", reviewCard.Card, reviewCard.Template, "2026-08-20T09:00:00Z", review.Good)
	two := makeAnswer("01B", reviewCard.Card, reviewCard.Template, "2026-08-21T09:00:00Z", review.Good)

	delta := review.NewFSRS()
	once := review.OrderAnswers([]review.Answer{one, two}).Replay(ahead, review.ScheduleBy(delta))
	// The whole run again, as a conflicted copy of the file lands beside it.
	twice := review.OrderAnswers([]review.Answer{one, two, one, two}).Replay(ahead, review.ScheduleBy(delta))

	if once[reviewCard] != twice[reviewCard] {
		t.Errorf("the same run twice left the card at %+v, once leaves it at %+v",
			twice[reviewCard], once[reviewCard])
	}
	if twice[reviewCard].Reps != 2 {
		t.Errorf("the card was answered twice and has been through %d", twice[reviewCard].Reps)
	}
}

// An answer taken back stays taken back however many copies of the line that
// took it back arrive.
func TestALineTakingAnAnswerBackTwiceConsumesItBackOnce(t *testing.T) {
	reviewCard := review.ReviewCardID{Card: "k7m2xq9fzp", Template: "Recognise"}
	provided := makeAnswer("01A", reviewCard.Card, reviewCard.Template, "2026-08-20T09:00:00Z", review.Good)
	back := review.Answer{ID: "01B", At: parseTime("2026-08-20T09:01:00Z"), Undoes: provided.ID}

	remaining := review.OrderAnswers([]review.Answer{provided, back, provided, back}).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if _, target := remaining[reviewCard]; target {
		t.Errorf("the answer was taken back and the card is at %+v", remaining[reviewCard])
	}
}

// means by a new card.
func TestReplay_UnansweredCardHasNoSchedule(t *testing.T) {
	remaining := review.OrderAnswers(nil).Replay(ahead, review.ScheduleBy(review.NewFSRS()))
	if len(remaining) != 0 {
		t.Errorf("no answers left %d schedules", len(remaining))
	}
	if !(review.Schedule{}).IsNew() {
		t.Error("unanswered schedule marked as not new")
	}
}

// session generates a vault's answers: a few review cards answered over a few
// weeks, some of them in the same millisecond, each answer under an identifier
// of its own.
//
// here as they are minted anywhere: one to an answer, and no two alike.
func session(t *rapid.T) []review.Answer {
	cards := rapid.SliceOfNDistinct(
		rapid.StringMatching(`[0-9abcdefghjkmnpqrstvwxyz]{10}`), 1, 3,
		func(s string) string { return s },
	).Draw(t, "cards")
	templates := rapid.SliceOfNDistinct(
		rapid.SampledFrom([]string{"Recognise", "Recall", "Spell", "Слово"}), 1, 3,
		func(s string) string { return s },
	).Draw(t, "templates")

	baseTime := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	answer := rapid.Custom(func(t *rapid.T) review.Answer {
		return review.Answer{
			ReviewCard: review.ReviewCardID{
				Card:     rapid.SampledFrom(cards).Draw(t, "card"),
				Template: rapid.SampledFrom(templates).Draw(t, "template"),
			},
			// Minutes over a few days, so two answers at one instant are generated about as
			// often as two at different instants.
			At:     baseTime.Add(time.Duration(rapid.IntRange(0, 5000).Draw(t, "minutes")) * time.Minute),
			Rating: review.Rating(rapid.IntRange(int(review.Again), int(review.Easy)).Draw(t, "rating")),
		}
	})

	answers := rapid.SliceOfN(answer, 1, 40).Draw(t, "answers")
	for item := range answers {
		answers[item].ID = fmt.Sprintf("01K3ZQ7X%018d", item)
	}
	return answers
}

// A schedule depends on the answers themselves and never on the order in which
// the files carrying them arrived, and one identifier is one answer however many
// lines carry it. A synchroniser that meets a conflict leaves a second copy of a
// run next to the first, and a restored backup can add another; a card counted
// twice is scheduled further out than it earned.
func TestASchedulesDoesNotDependOnHowTheRunsWereDelivered(t *testing.T) {
	t.Parallel()
	var copied, level int
	rapid.Check(t, func(t *rapid.T) {
		answers := session(t)
		// Two answers at one instant are ordered by their identifiers, which is the only
		// place the order is not the clock's.
		// The instant is held as a count of nanoseconds: two time.Time values naming one
		// instant are two map keys, and this checks whether they are one.
		when := make(map[review.ReviewCardID]map[int64]bool)
		for _, a := range answers {
			if when[a.ReviewCard] == nil {
				when[a.ReviewCard] = make(map[int64]bool)
			}
			if when[a.ReviewCard][a.At.UnixNano()] {
				level++
				break
			}
			when[a.ReviewCard][a.At.UnixNano()] = true
		}

		// The runs are carried in whatever order, and any of them may arrive
		// twice.
		as := make([]review.Answer, 0, 2*len(answers))
		for _, item := range rapid.Permutation(order(len(answers))).Draw(t, "carried") {
			as = append(as, answers[item])
			if rapid.Bool().Draw(t, "twice") {
				as = append(as, answers[item])
				copied++
			}
		}

		delta := review.NewFSRS()
		want := review.OrderAnswers(answers).Replay(ahead, review.ScheduleBy(delta))
		got := review.OrderAnswers(as).Replay(ahead, review.ScheduleBy(delta))
		if !maps.Equal(got, want) {
			t.Fatalf("%d answers delivered as %d gave %+v, want %+v",
				len(answers), len(as), got, want)
		}
	})
	// A run that carried nothing twice, and a history no two answers of which
	// stand at one instant, ask nothing of either rule.
	if copied < 100 || level < 5 {
		t.Fatalf("%d answers were delivered twice, over %d histories holding "+
			"two answers of one review card at one instant", copied, level)
	}
}

// An answer taken back leaves the schedule that the surrounding answers produce
// on their own: an answer that was taken back is not counted as given, whatever
// else the file holds and however many lines take it back.
func TestAnAnswerTakenBackPreservesTheScheduleItWasNeverGivenIn(t *testing.T) {
	t.Parallel()
	var took, twice int
	rapid.Check(t, func(t *rapid.T) {
		answers := session(t)

		expectedAnswers := make([]review.Answer, 0, len(answers))
		with := make([]review.Answer, 0, 2*len(answers))
		for _, a := range answers {
			with = append(with, a)
			if !rapid.Bool().Draw(t, "taken back") {
				expectedAnswers = append(expectedAnswers, a)
				continue
			}
			took++
			// A line taking an answer back is placed after the answer and before whatever
			// followed.
			lines := 1
			if rapid.Bool().Draw(t, "taken back twice") {
				lines, twice = 2, twice+1
			}
			for i := range lines {
				with = append(with, review.Answer{
					ID:     fmt.Sprintf("%s-undo-%d", a.ID, i),
					At:     a.At.Add(time.Second),
					Undoes: a.ID,
				})
			}
		}

		delta := review.NewFSRS()
		want := review.OrderAnswers(expectedAnswers).Replay(ahead, review.ScheduleBy(delta))
		got := review.OrderAnswers(with).Replay(ahead, review.ScheduleBy(delta))
		if !maps.Equal(got, want) {
			t.Fatalf("%d of %d answers taken back gave %+v, want %+v",
				took, len(answers), got, want)
		}
	})
	// A run in which nothing was taken back does not exercise the rule.
	if took < 200 || twice < 50 {
		t.Fatalf("%d answers were taken back and %d of those twice over", took, twice)
	}
}

// order is the places of a history, for a permutation to be drawn over.
func order(n int) []int {
	out := make([]int, n)
	for item := range out {
		out[item] = item
	}
	return out
}
