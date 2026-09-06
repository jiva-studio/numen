package review_test

import (
	"fmt"
	"maps"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// session generates a vault's answers: a few card faces answered over a few
// weeks, some of them in the same millisecond, each answer under an identifier
// of its own.
//
// The identifiers are what a history is held together by, so they are minted
// here as they are minted anywhere: one to an answer, and no two alike.
func session(t *rapid.T) []review.Answer {
	cards := rapid.SliceOfNDistinct(
		rapid.StringMatching(`[0-9abcdefghjkmnpqrstvwxyz]{10}`), 1, 3,
		func(s string) string { return s },
	).Draw(t, "cards")
	faces := rapid.SliceOfNDistinct(
		rapid.SampledFrom([]string{"Recognise", "Recall", "Spell", "Слово"}), 1, 3,
		func(s string) string { return s },
	).Draw(t, "faces")

	opened := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	answer := rapid.Custom(func(t *rapid.T) review.Answer {
		return review.Answer{
			CardFace: review.CardFaceID{
				Card: rapid.SampledFrom(cards).Draw(t, "card"),
				Face: rapid.SampledFrom(faces).Draw(t, "face"),
			},
			// Minutes, over a few weeks, so that two answers of one instant are
			// drawn about as often as two of different ones.
			At:     opened.Add(time.Duration(rapid.IntRange(0, 30000).Draw(t, "minutes")) * time.Minute),
			Rating: review.Rating(rapid.IntRange(int(review.Again), int(review.Easy)).Draw(t, "rating")),
		}
	})

	answers := rapid.SliceOfN(answer, 1, 40).Draw(t, "answers")
	for at := range answers {
		answers[at].ID = fmt.Sprintf("01K3ZQ7X%018d", at)
	}
	return answers
}

// A schedule is the answers themselves and never the order the files carrying
// them arrived in, and one identifier is one answer however many lines carry
// it. A synchroniser that met a conflict leaves a second copy of a run beside
// the first, and a person restoring a backup puts one there by hand; a card
// counted twice is sent away for longer than it was earned.
func TestASchedulesDoesNotDependOnHowTheRunsWereDelivered(t *testing.T) {
	t.Parallel()
	var copied, level int
	rapid.Check(t, func(t *rapid.T) {
		answers := session(t)
		// Two answers of one instant are put in the order of their identifiers,
		// which is the one place the order is not the clock's.
		when := make(map[review.CardFaceID]map[time.Time]bool)
		for _, a := range answers {
			if when[a.CardFace] == nil {
				when[a.CardFace] = make(map[time.Time]bool)
			}
			if when[a.CardFace][a.At] {
				level++
				break
			}
			when[a.CardFace][a.At] = true
		}

		// The runs are carried in whatever order, and any of them may arrive
		// twice.
		as := make([]review.Answer, 0, 2*len(answers))
		for _, at := range rapid.Permutation(order(len(answers))).Draw(t, "carried") {
			as = append(as, answers[at])
			if rapid.Bool().Draw(t, "twice") {
				as = append(as, answers[at])
				copied++
			}
		}

		by := review.NewFSRS()
		want := review.Replay(ahead, by, answers)
		got := review.Replay(ahead, by, as)
		if !maps.Equal(got, want) {
			t.Fatalf("%d answers delivered as %d gave %+v, want %+v",
				len(answers), len(as), got, want)
		}
	})
	// A run that carried nothing twice, and a history no two answers of which
	// stand at one instant, ask nothing of either rule.
	if copied < 100 || level < 5 {
		t.Fatalf("%d answers were delivered twice, over %d histories holding "+
			"two answers of one card face at one instant", copied, level)
	}
}

// An answer taken back leaves the schedule the answers around it would have
// left on their own: a person who took one back is not counted as having given
// it, whatever else the file holds and however many lines take it back.
func TestAnAnswerTakenBackLeavesTheScheduleItWasNeverGivenIn(t *testing.T) {
	t.Parallel()
	var took, twice int
	rapid.Check(t, func(t *rapid.T) {
		answers := session(t)

		kept := make([]review.Answer, 0, len(answers))
		with := make([]review.Answer, 0, 2*len(answers))
		for _, a := range answers {
			with = append(with, a)
			if !rapid.Bool().Draw(t, "taken back") {
				kept = append(kept, a)
				continue
			}
			took++
			// A line taking an answer back stands where the person pressed it,
			// which is after the answer and before whatever followed.
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

		by := review.NewFSRS()
		want := review.Replay(ahead, by, kept)
		got := review.Replay(ahead, by, with)
		if !maps.Equal(got, want) {
			t.Fatalf("%d of %d answers taken back gave %+v, want %+v",
				took, len(answers), got, want)
		}
	})
	// A run in which nothing was taken back asks nothing of the rule.
	if took < 200 || twice < 50 {
		t.Fatalf("%d answers were taken back and %d of those twice over", took, twice)
	}
}

// order is the places of a history, for a permutation to be drawn over.
func order(n int) []int {
	out := make([]int, n)
	for at := range out {
		out[at] = at
	}
	return out
}
