package review_test

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// The four ratings order the next due day: the better the rating, the later the
// card is due.
func TestTheBetterACardWasRecalledTheLongerItIsLeft(t *testing.T) {
	delta := review.NewFSRS()
	when := parseTime(t, "2026-08-29T09:00:00Z")

	// A card already in review, so the four ratings are distinguished by their effect
	// on a scheduled card and not by the first learning steps.
	learned := delta.Next(delta.Next(review.Schedule{}, when, review.Good),
		when.Add(10*24*time.Hour), review.Good)
	later := learned.Due

	var last time.Time
	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		due := delta.Next(learned, later, r).Due
		if !due.After(last) {
			t.Errorf("%v is due %v, which is no later than the answer before it at %v", r, due, last)
		}
		last = due
	}
}

// A card that was not recalled at all is a lapse, and it is counted.
func TestACardThatWasNotRecalledIsALapse(t *testing.T) {
	delta := review.NewFSRS()
	when := parseTime(t, "2026-08-29T09:00:00Z")

	learned := delta.Next(delta.Next(review.Schedule{}, when, review.Good),
		when.Add(10*24*time.Hour), review.Good)
	if learned.Lapses != 0 {
		t.Fatalf("a card answered well twice has %d lapses", learned.Lapses)
	}

	cleared := delta.Next(learned, learned.Due, review.Again)
	if cleared.Lapses != 1 {
		t.Errorf("a card that did not come back has %d lapses, want 1", cleared.Lapses)
	}
	if cleared.Reps != learned.Reps+1 {
		t.Errorf("answers counted %d, want %d", cleared.Reps, learned.Reps+1)
	}
}

// An answer marks a card as reviewed, which distinguishes it from a new card.
func TestFSRSNext_ReviewedCardIsNotNew(t *testing.T) {
	delta := review.NewFSRS()
	when := parseTime(t, "2026-08-29T09:00:00Z")

	if delta.Next(review.Schedule{}, when, review.Good).IsNew() {
		t.Error("a card answered once is still new")
	}
	if got := delta.Next(review.Schedule{}, when, review.Good).Last; !got.Equal(when) {
		t.Errorf("answered at %v, recorded at %v", when, got)
	}
}

// A scheduler carries nothing from one answer to the next, so one answers for many review cards
// at once, each answer as it would be on its own. Under the race detector this holds it to
// keeping no state: a projection of a preset shares one scheduler across every place of a curve.
func TestASchedulerAnswersForManyReviewCardsAtOnce(t *testing.T) {
	delta := review.NewFSRS()
	when := parseTime(t, "2026-08-29T09:00:00Z")

	// still putting into memory, and ones it has put into review.
	cards := []review.Schedule{{}}
	for _, r := range []review.Rating{review.Again, review.Good, review.Easy} {
		one := delta.Next(review.Schedule{}, when.Add(-30*24*time.Hour), r)
		cards = append(cards, one, delta.Next(one, one.Due, review.Good))
	}

	alone := make([]review.Schedule, len(cards))
	good := make([]review.Schedule, len(cards))
	again := make([]review.Schedule, len(cards))
	for i, c := range cards {
		alone[i] = delta.Next(c, when, review.Good)
		good[i], again[i] = delta.Endings(c, when)
	}

	// Every asking writes its own answer down, so what the detector sees is the
	// scheduler being shared and nothing this test does with the answers.
	const rounds = 8
	pos := make([][]review.Schedule, rounds)
	ends := make([][]review.Schedule, rounds)
	fell := make([][]review.Schedule, rounds)
	var together sync.WaitGroup
	for round := range rounds {
		pos[round] = make([]review.Schedule, len(cards))
		ends[round] = make([]review.Schedule, len(cards))
		fell[round] = make([]review.Schedule, len(cards))
		for i, c := range cards {
			together.Go(func() {
				pos[round][i] = delta.Next(c, when, review.Good)
				ends[round][i], fell[round][i] = delta.Endings(c, when)
			})
		}
	}
	together.Wait()

	for round := range rounds {
		for i := range cards {
			if pos[round][i] != alone[i] {
				t.Errorf("review card %d answered %v at once and %v alone",
					i, pos[round][i], alone[i])
			}
			if ends[round][i] != good[i] || fell[round][i] != again[i] {
				t.Errorf("review card %d ends %v and %v at once, %v and %v alone",
					i, ends[round][i], fell[round][i], good[i], again[i])
			}
		}
	}
}

// A schedule in review with no stability of its own still answers with a
// finite, positive stability, because data a scheduler never wrote can carry
// a review phase without one.
func TestAReviewScheduleWithNoStabilityAnswersWithoutPoisoning(t *testing.T) {
	delta := review.NewFSRS()
	when := parseTime(t, "2026-08-29T09:00:00Z")

	corrupt := review.Schedule{
		Last:      when.Add(-10 * 24 * time.Hour),
		Phase:     uint8(fsrs.Review),
		Stability: 0,
	}

	assertFinitePositive := func(t *testing.T, label string, got float64) {
		t.Helper()
		if math.IsNaN(got) || math.IsInf(got, 0) || got <= 0 {
			t.Errorf("%s left stability %v, want a finite positive number", label, got)
		}
	}

	for _, r := range []review.Rating{review.Again, review.Hard, review.Good, review.Easy} {
		assertFinitePositive(t, r.String(), delta.Next(corrupt, when, r).Stability)
	}

	good, again := delta.Endings(corrupt, when)
	assertFinitePositive(t, "Endings good", good.Stability)
	assertFinitePositive(t, "Endings again", again.Stability)
}

// A schedule records which scheduler produced it, because the values carried
// between answers are specific to that scheduler.
func TestASchedulerReturnsWhichItIs(t *testing.T) {
	got := review.NewFSRS().GetName()
	if !strings.HasPrefix(got, review.FSRSName+".") {
		t.Errorf("named itself %q, want the algorithm and what it is running on", got)
	}
	if got == review.FSRSName+"." {
		t.Error("named itself the algorithm and nothing about its parameters")
	}
	if again := review.NewFSRS().GetName(); again != got {
		t.Errorf("named itself %q and then %q", got, again)
	}
}

func getLibraryNow() time.Time {
	return time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)
}

func getLibraryGaps() []time.Duration {
	return []time.Duration{
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
}

func getLibraryRatings() []fsrs.Rating {
	return []fsrs.Rating{fsrs.Again, fsrs.Hard, fsrs.Good, fsrs.Easy}
}

func getLibraryRetentions() []float64 {
	return []float64{0, 0.7, 0.85, 0.87, 0.9, 0.97, 0.99}
}

// The scheduler works an interval out from its parameters, and the result is what the library
// those parameters come from works out. Every field of a schedule is compared bit for bit, not
// within a tolerance, since a schedule differing in the last place can send a review card away
// for a different day. Coverage: sequences across review phases from new cards, and synthetic
// boundary schedules for clamping limits.
func TestASchedulerWorksOutWhatTheLibraryWorksOut(t *testing.T) {
	for _, retention := range getLibraryRetentions() {
		delta, engine := makeSchedulers(retention)
		for _, s := range libraryWalked(engine) {
			libraryAgrees(t, retention, delta, engine, s)
		}
		for _, s := range libraryMade() {
			libraryAgrees(t, retention, delta, engine, s)
		}
	}
}

func makeSchedulers(retention float64) (review.FSRS, *fsrs.FSRS) {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	if retention == 0 {
		return review.NewFSRS(), fsrs.NewFSRS(p)
	}
	p.RequestRetention = math.Min(math.Max(retention,
		review.GetRetentionBounds().Min), review.GetRetentionBounds().Max)
	return review.NewFSRSAt(retention), fsrs.NewFSRS(p)
}

func libraryAgrees(
	t *testing.T, retention float64, delta review.FSRS, engine *fsrs.FSRS,
	s review.Schedule,
) {
	t.Helper()
	// A review schedule with zero stability is floored before it is read. The
	// library's result for it is not compared here; its own test checks the
	// scheduler's handling.
	if fsrs.State(s.Phase) == fsrs.Review && s.Stability <= 0 {
		return
	}
	status := clampLastAnswer(s)
	for _, r := range getLibraryRatings() {
		want := asSchedule(engine.Next(asCard(status), getLibraryNow(), r).Card)
		got := delta.Next(s, getLibraryNow(), review.Rating(r))
		sameSchedule(t, fmt.Sprintf("at %.2f, %s of %s", retention, r, describe(s)), got, want)
	}
	good, again := delta.Endings(s, getLibraryNow())
	sameSchedule(t, fmt.Sprintf("at %.2f, the ending it came back on of %s", retention, describe(s)),
		good, asSchedule(engine.Next(asCard(status), getLibraryNow(), fsrs.Good).Card))
	sameSchedule(t, fmt.Sprintf("at %.2f, the ending it did not of %s", retention, describe(s)),
		again, asSchedule(engine.Next(asCard(status), getLibraryNow(), fsrs.Again).Card))
}

func clampLastAnswer(s review.Schedule) review.Schedule {
	if !s.IsNew() && getLibraryNow().Before(s.Last) {
		s.Last = getLibraryNow()
	}
	return s
}

// A review card whose last answer is not before the evaluation instant has zero
// elapsed days, however far after that instant the answer falls.
//
// Elapsed days are never negative, so every such card yields the result the
// library gives a card whose last answer falls at that instant.
func TestFSRSNext_ClampsElapsedDaysToZero(t *testing.T) {
	delta, engine := makeSchedulers(0)

	// A review card in the review phase whose last answer is after the evaluation
	// instant.
	ahead := delta.Next(delta.Next(review.Schedule{},
		getLibraryNow().AddDate(0, 0, -30), review.Good),
		getLibraryNow().AddDate(0, 0, 1), review.Good)
	if fsrs.State(ahead.Phase) != fsrs.Review {
		t.Fatalf("review card phase = %d, want %d", ahead.Phase, fsrs.Review)
	}

	// The same review card with its last answer at the evaluation instant.
	status := ahead
	status.Last = getLibraryNow()

	for _, r := range getLibraryRatings() {
		want := asSchedule(engine.Next(asCard(status), getLibraryNow(), r).Card)
		for _, dur := range []time.Duration{
			time.Second, time.Hour, 24 * time.Hour, 400 * 24 * time.Hour,
		} {
			reviewCard := ahead
			reviewCard.Last = getLibraryNow().Add(dur)
			got := delta.Next(reviewCard, getLibraryNow(), review.Rating(r))
			sameSchedule(t, fmt.Sprintf("%s of a review card answered %s ahead", r, dur), got, want)
		}
	}
}

func libraryWalked(engine *fsrs.FSRS) []review.Schedule {
	type step struct {
		s         review.Schedule
		timePoint time.Time
	}
	out := []review.Schedule{{}}
	frontier := []step{{timePoint: getLibraryNow().Add(-getLibraryWalkOpens())}}
	for depth := range 7 {
		var next []step
		for _, f := range frontier {
			for i, r := range getLibraryRatings() {
				pos := f.timePoint.Add(getLibraryGaps()[(depth+i)%len(getLibraryGaps())])
				one := asSchedule(engine.Next(asCard(f.s), pos, r).Card)
				out = append(out, one)
				next = append(next, step{one, pos})
			}
		}
		frontier = next
	}
	return out
}

func getLibraryWalkOpens() time.Duration {
	var all time.Duration
	for _, g := range getLibraryGaps() {
		all += g
	}
	return all
}

func libraryMade() []review.Schedule {
	stabilities := []float64{
		0, 1e-6, review.LeastStability, 0.01, 0.1, 1, 2.5, 7, 20, 100, 1000, 36500, 1e6,
	}
	difficulties := []float64{0, 1, 1.5, 3.2, 5, 7.7, 10}
	// Negative gaps give a review card whose last answer is after the evaluation
	// instant. A projection produces such cards: the material it starts from carries
	// answers whose interval extends past the start day.
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
						Due:        getLibraryNow().Add(time.Duration(j) * 24 * time.Hour),
						Last:       getLibraryNow().Add(-gap),
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

func describe(s review.Schedule) string {
	return fmt.Sprintf("a review card in phase %d at stability %v difficulty %v answered %v",
		s.Phase, s.Stability, s.Difficulty, s.Last)
}
