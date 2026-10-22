package review_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
)

// libraryNow is the instant the schedules of this file are answered at.
var libraryNow = time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)

// libraryGaps are how long a card face stood away before it was answered: an
// answer given at once, one given in the minutes a card is put into memory
// over, and one given days, months and years later.
var libraryGaps = []time.Duration{
	0,
	time.Minute,
	10 * time.Minute,
	time.Hour,
	23 * time.Hour,
	24 * time.Hour,
	3 * 24 * time.Hour,
	17 * 24 * time.Hour,
	200 * 24 * time.Hour,
	3 * 365 * 24 * time.Hour,
}

// libraryRatings are the four answers, in the library's own vocabulary.
var libraryRatings = []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy}

// libraryRetentions are the shares of the cards a scheduler may be asked to
// bring back: the scheduler on its published parameters, and one at each end of
// what a preset may hold and a few places between.
var libraryRetentions = []float64{0, 0.7, 0.85, 0.87, 0.9, 0.97, 0.99}

// The scheduler works an interval out from its parameters, and what it works
// out is what the library those parameters come from works out.
//
// Every field of a schedule is compared bit for bit and not within a tolerance:
// a schedule differing in the last place is a different schedule, and a card
// face sent away for a different day.
//
// Test coverage includes sequences generated across review phases from new
// cards, and synthetic boundary schedules covering clamping limits.
func TestASchedulerWorksOutWhatTheLibraryWorksOut(t *testing.T) {
	for _, retention := range libraryRetentions {
		delta, engine := makeSchedulers(retention)
		for _, s := range libraryWalked(engine) {
			libraryAgrees(t, retention, delta, engine, s)
		}
		for _, s := range libraryMade() {
			libraryAgrees(t, retention, delta, engine, s)
		}
	}
}

// makeSchedulers is the scheduler and the library it takes its parameters from,
// both asking for the same share of the cards. A retention of nothing is the
// scheduler on its published parameters.
func makeSchedulers(retention float64) (review.FSRS, *fsrs.FSRS) {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	if retention == 0 {
		return review.NewFSRS(), fsrs.NewFSRS(p)
	}
	p.RequestRetention = math.Min(math.Max(retention,
		review.RetentionBounds.Least), review.RetentionBounds.Most)
	return review.NewFSRSAt(retention), fsrs.NewFSRS(p)
}

// libraryAgrees asks both about one card face, at every rating and at both
// endings, and holds the two answers to being the same schedule.
//
// What the library is asked about is the card face as it stood away. A card
// face answered no later than the answer it already carries stood no time away,
// and that is the card face the library is asked about — the same arithmetic on
// the days that actually went by, and what the scheduler makes of such a card
// face is written down in the test below.
func libraryAgrees(
	t *testing.T, retention float64, delta review.FSRS, engine *fsrs.FSRS,
	s review.Schedule,
) {
	t.Helper()
	// A review schedule standing at no stability is floored before it is
	// read. The library's answer to one is not asked here; its own test
	// checks what this scheduler makes of it.
	if fsrs.State(s.Phase) == fsrs.Review && s.Stability <= 0 {
		return
	}
	status := clampLastAnswer(s)
	for _, r := range libraryRatings {
		want := asSchedule(engine.Next(asCard(status), libraryNow, r).Card)
		got := delta.Next(s, libraryNow, review.Rating(r))
		sameSchedule(t, fmt.Sprintf("at %.2f, %s of %s", retention, r, describe(s)), got, want)
	}
	good, again := delta.Endings(s, libraryNow)
	sameSchedule(t, fmt.Sprintf("at %.2f, the ending it came back on of %s", retention, describe(s)),
		good, asSchedule(engine.Next(asCard(status), libraryNow, fsrs.Good).Card))
	sameSchedule(t, fmt.Sprintf("at %.2f, the ending it did not of %s", retention, describe(s)),
		again, asSchedule(engine.Next(asCard(status), libraryNow, fsrs.Again).Card))
}

// clampLastAnswer is a card face at the days it stood away. One answered no
// later than the answer it already carries stood none of them.
func clampLastAnswer(s review.Schedule) review.Schedule {
	if !s.IsNew() && libraryNow.Before(s.Last) {
		s.Last = libraryNow
	}
	return s
}

// A card face answered no later than the answer it already carries stood no
// time away, however far ahead of the instant it is asked about that answer is.
//
// The days a card face stood away do not go below none, so every such card face
// is answered the same, and answered as the library answers one whose last
// answer falls at that instant.
func TestACardFaceAnsweredNoLaterThanItsLastAnswerStoodNoTimeAway(t *testing.T) {
	delta, engine := makeSchedulers(0)

	// A card face the scheduler has put into review, carrying an answer given
	// after the instant it is asked about.
	ahead := delta.Next(delta.Next(review.Schedule{},
		libraryNow.AddDate(0, 0, -30), review.Good),
		libraryNow.AddDate(0, 0, 1), review.Good)
	if fsrs.State(ahead.Phase) != fsrs.Review {
		t.Fatalf("card face phase = %d, want %d", ahead.Phase, fsrs.Review)
	}

	// The same card face as one that stood no time away.
	status := ahead
	status.Last = libraryNow

	for _, r := range libraryRatings {
		want := asSchedule(engine.Next(asCard(status), libraryNow, r).Card)
		for _, dur := range []time.Duration{
			time.Second, time.Hour, 24 * time.Hour, 400 * 24 * time.Hour,
		} {
			face := ahead
			face.Last = libraryNow.Add(dur)
			got := delta.Next(face, libraryNow, review.Rating(r))
			sameSchedule(t, fmt.Sprintf("%s of a card face answered %s ahead", r, dur), got, want)
		}
	}
}

// libraryWalked is every card face a walk of seven answers reaches, each answer
// given a gap of its own after the answer already on the card face.
//
// The walk opens far enough back that its last answer still falls before the
// instant these card faces are asked about, so every gap the library is handed
// runs forwards.
func libraryWalked(engine *fsrs.FSRS) []review.Schedule {
	type step struct {
		s  review.Schedule
		at time.Time
	}
	out := []review.Schedule{{}}
	frontier := []step{{at: libraryNow.Add(-libraryWalkOpens)}}
	for depth := range 7 {
		var next []step
		for _, f := range frontier {
			for i, r := range libraryRatings {
				pos := f.at.Add(libraryGaps[(depth+i)%len(libraryGaps)])
				one := asSchedule(engine.Next(asCard(f.s), pos, r).Card)
				out = append(out, one)
				next = append(next, step{one, pos})
			}
		}
		frontier = next
	}
	return out
}

// libraryWalkOpens is how long before the instant a card face is asked about a
// walk of seven answers opens. Every gap taken together is more than seven of
// them, so no answer of the walk falls after that instant.
var libraryWalkOpens = func() time.Duration {
	var all time.Duration
	for _, g := range libraryGaps {
		all += g
	}
	return all
}()

// libraryMade is a grid of card faces standing where a walk does not reach: at
// the stability a run of lapses wears a card down to and at the stability of a
// card nothing shifts, at each end of the difficulty scale, and answered after
// gaps from none to years.
func libraryMade() []review.Schedule {
	stabilities := []float64{
		0, 1e-6, review.LeastStability, 0.01, 0.1, 1, 2.5, 7, 20, 100, 1000, 36500, 1e6,
	}
	difficulties := []float64{0, 1, 1.5, 3.2, 5, 7.7, 10}
	// A card face answered before the answer it already carries stands a
	// negative number of days away, which a projection reaches: the material it
	// opens on carries answers whose interval runs past the day it opens.
	gaps := []time.Duration{
		-400 * 24 * time.Hour, -40 * 24 * time.Hour, -24 * time.Hour, -time.Minute,
		0, time.Minute, 24 * time.Hour, 2 * 24 * time.Hour,
		30 * 24 * time.Hour, 365 * 24 * time.Hour, 4000 * 24 * time.Hour,
	}
	phases := []fsrs.State{fsrs.New, fsrs.Learning, fsrs.Review, fsrs.Relearning}

	var out []review.Schedule
	for _, stability := range stabilities {
		for _, difficulty := range difficulties {
			for i, gap := range gaps {
				for j, phase := range phases {
					out = append(out, review.Schedule{
						Due:        libraryNow.Add(time.Duration(j) * 24 * time.Hour),
						Last:       libraryNow.Add(-gap),
						Reps:       1 + i*13 + j,
						Lapses:     j * 3,
						Stability:  stability,
						Difficulty: difficulty,
						Phase:      uint8(phase),
					})
				}
			}
		}
	}
	return out
}

// sameSchedule holds two answers to being the same schedule, field by field and
// bit for bit.
func sameSchedule(t *testing.T, what string, got, want review.Schedule) {
	t.Helper()
	if !got.Due.Equal(want.Due) {
		t.Fatalf("%s comes round at %v, want %v", what, got.Due, want.Due)
	}
	if !got.Last.Equal(want.Last) {
		t.Fatalf("%s was answered at %v, want %v", what, got.Last, want.Last)
	}
	if got.Reps != want.Reps {
		t.Fatalf("%s has had %d answers, want %d", what, got.Reps, want.Reps)
	}
	if got.Lapses != want.Lapses {
		t.Fatalf("%s has %d lapses, want %d", what, got.Lapses, want.Lapses)
	}
	if math.Float64bits(got.Stability) != math.Float64bits(want.Stability) {
		t.Fatalf("%s stability = %v, want %v", what, got.Stability, want.Stability)
	}
	if math.Float64bits(got.Difficulty) != math.Float64bits(want.Difficulty) {
		t.Fatalf("%s difficulty = %v, want %v", what, got.Difficulty, want.Difficulty)
	}
	if got.Phase != want.Phase {
		t.Fatalf("%s is left in phase %d, want %d", what, got.Phase, want.Phase)
	}
}

// asCard converts review.Schedule into an FSRS library Card model.
func asCard(s review.Schedule) fsrs.Card {
	if s.IsNew() {
		return fsrs.NewCard()
	}
	return fsrs.Card{
		Due:        s.Due,
		Stability:  s.Stability,
		Difficulty: s.Difficulty,
		Reps:       uint64(s.Reps),
		Lapses:     uint64(s.Lapses),
		State:      fsrs.State(s.Phase),
		LastReview: s.Last,
	}
}

// asSchedule is where the library's card stands, as a schedule.
func asSchedule(c fsrs.Card) review.Schedule {
	return review.Schedule{
		Due:        c.Due,
		Last:       c.LastReview,
		Reps:       int(c.Reps),
		Lapses:     int(c.Lapses),
		Stability:  c.Stability,
		Difficulty: c.Difficulty,
		Phase:      uint8(c.State),
	}
}

// describe is one card face, as much of it as names which card face it was.
func describe(s review.Schedule) string {
	return fmt.Sprintf("a card face in phase %d at stability %v difficulty %v answered %v",
		s.Phase, s.Stability, s.Difficulty, s.Last)
}
