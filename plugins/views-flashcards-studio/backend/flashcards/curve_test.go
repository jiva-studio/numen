package flashcards_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review/projection"
)

// newStudiedNotes is a vault of one preset, one deck of as many cards pointing
// which is what a person meets the first time they open one of these controls.
func newStudiedNotes(cards int) map[string]string {
	deck := "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n---\n"
	for i := range cards {
		deck += fmt.Sprintf(
			"\n## Word %d ^card%06d\n\n[[Term]]\n\n### Word\n\nWord %d\n\n### Meaning\n\nMeaning %d\n",
			i, i, i, i)
	}
	return map[string]string{
		"Term.md": vault["Term.md"],
		"Sanskrit.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
			"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\n---\n\n# Sanskrit\n",
		"decks/Roots.md": deck,
		"decks/Other.md": "---\ntype: deck\n---\n" +
			"\n## Leaf mould ^3f4g5h6j7k\n\n[[Term]]\n\n### Word\n\nLeaf mould\n\n### Meaning\n\nLeaves\n",
	}
}

// newAnsweredVault is that vault with a history behind it: a dozen of its card
// review cards answered three times each in the months before the day the curves are
// drawn on, and the rest of them unbegun.
//
// A curve is computed from past review costs in this vault.
func newAnsweredVault(t *testing.T, cards int) vaulted {
	t.Helper()
	s := openVault(t, newStudiedNotes(cards))
	for _, days := range []int{150, 120, 90} {
		record := s.run(t, noon.AddDate(0, 0, -days))
		for i := range min(cards, 12) {
			took := 4 * time.Second
			if i%7 == 0 {
				took = 50 * time.Second
			}
			answer(t, record, cardID(i), took)
		}
	}
	return s
}

// curves is the simulator over one vault, on a named day.
func (s vaulted) curves(now time.Time) flashcards.ProjectCurve {
	u := flashcards.NewProjectCurve(
		s.standings, s.schedules, s.presets,
		today, func() time.Time { return now },
	)
	u.Cores = runtime.GOMAXPROCS(0)
	return u
}

// noon is a fixed hour, so that a curve is the same curve whenever the tests
// are run.
var noon = time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)

// A curve is asked for once per position of a control a person is dragging, and
// none of those asks writes anything.
func TestACurveWritesNoScheduleCache(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	if _, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p); err != nil {
		t.Fatal(err)
	}
	if raw, err := s.schedules.Cache.Read(t.Context(), s.vault.ID); err == nil {
		t.Errorf("a curve wrote a cache of %d bytes", len(raw))
	}
}

// The whole range of the goal is worked out in one pass: a control moving over
// it reads a finished array and computes nothing.
func TestTheCurveOfMinutesCoversTheWholeRange(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != flashcards.Points || len(got.Points) != flashcards.Points {
		t.Fatalf("the curve has %d places and %d values", len(got.Grid), len(got.Points))
	}
	for i := 1; i < len(got.Grid); i++ {
		if got.Grid[i] <= got.Grid[i-1] {
			t.Fatalf("the grid runs %v then %v", got.Grid[i-1], got.Grid[i])
		}
		if got.Points[i].Reviews < got.Points[i-1].Reviews {
			t.Errorf("a longer day answers %v, a shorter one %v",
				got.Points[i].Reviews, got.Points[i-1].Reviews)
		}
		if got.Points[i].Owed > got.Points[i-1].Owed {
			t.Errorf("a longer day leaves %d owed, a shorter one %d",
				got.Points[i].Owed, got.Points[i-1].Owed)
		}
	}
	inRange(t, got, "now", "suggested")
	if got.Points[len(got.Points)-1].Owed != 0 {
		t.Errorf("the longest day on the grid still leaves %d owed", got.Points[len(got.Points)-1].Owed)
	}
}

// What a curve of minutes suggests is the first place of the grid the minutes
// no longer close: the shortest day that asks everything the day holds.
func TestTheShortestDayThatAsksEverythingIsSuggested(t *testing.T) {
	t.Parallel()
	// A vault deep enough that the short days of the grid are cut short by the
	// clock and the long ones are not.
	s := newAnsweredVault(t, 200)
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	want := flashcards.OutsideGrid
	for i, one := range got.Points {
		if len(one.Closed) == 0 {
			want = flashcards.CurvePoint{Index: i, Value: got.Grid[i]}
			break
		}
	}
	if want.Index <= 0 || want.Index >= len(got.Points)-1 {
		t.Fatalf("the first place asking everything the day holds is %d of %d",
			want.Index, len(got.Points))
	}
	if got.Suggested != want {
		t.Errorf("%+v is suggested, and the shortest day asking everything is %+v",
			got.Suggested, want)
	}
	// And the place before it is one the minutes closed.
	if before := got.Points[want.Index-1]; !before.Closed.Has(review.ClosedMinutes) {
		t.Errorf("the place under the one suggested was closed by %v", before.Closed)
	}
}

// Higher target retention increases daily review requirements.
func TestTheCurveOfRetentionCoversTheWholeRange(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalRetention, Retention: 0.87, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != flashcards.Points {
		t.Fatalf("the curve has %d places", len(got.Grid))
	}
	if got.Grid[0] != review.RetentionBounds.Min ||
		got.Grid[len(got.Grid)-1] != review.RetentionBounds.Max {
		t.Errorf("the grid runs from %v to %v", got.Grid[0], got.Grid[len(got.Grid)-1])
	}
	for i := 1; i < len(got.Points); i++ {
		if got.Points[i].Minutes < got.Points[i-1].Minutes-1e-9 {
			t.Errorf("a target of %v costs %v minutes, one of %v costs %v",
				got.Grid[i], got.Points[i].Minutes, got.Grid[i-1], got.Points[i-1].Minutes)
		}
	}
	inRange(t, got, "now")

	if got.Suggested != flashcards.OutsideGrid {
		t.Errorf("a target of %v is suggested, and this goal points at none",
			got.Suggested.Value)
	}
	// What the cost buys climbs the whole range, and peaks at the far end of it.
	least, most := got.Points[0].Retained, got.Points[len(got.Points)-1].Retained
	if most <= least {
		t.Errorf("the easiest target retains %v and the hardest %v", least, most)
	}
}

// A date goal range runs from tomorrow past the target date.
func TestTheCurveOfADateRunsPastTheDayNamed(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		// A review card sent away for three weeks, so the days near the left of
		// the range cannot learn one and the days to the right can.
		Rule: review.RuleInterval, Interval: 21, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != len(got.Days) || len(got.Grid) != len(got.Points) {
		t.Fatalf("%d places, %d days, %d values", len(got.Grid), len(got.Days), len(got.Points))
	}
	if len(got.Grid) < 2 {
		t.Fatalf("the curve has %d places", len(got.Grid))
	}
	// Tomorrow, because a date of today is no period at all.
	if got.Grid[0] != 1 {
		t.Errorf("the curve begins %v days off", got.Grid[0])
	}
	if last := got.Grid[len(got.Grid)-1]; last <= 20 {
		t.Errorf("the curve runs to %v days off, and the day named is 20 off", last)
	}
	if got.Now.Value != 20 || got.Now.Day != p.By.Format(review.DayLayout) {
		t.Errorf("got.Now = %+v, want value 20, day %s", got.Now, p.By.Format(review.DayLayout))
	}
	if got.Now.Index <= 0 || got.Now.Index >= len(got.Grid)-1 {
		t.Errorf("got.Now.Index = %d, want inside range (0, %d)",
			got.Now.Index, len(got.Grid)-1)
	}

	for i := 1; i < len(got.Points); i++ {
		was, now := got.Points[i-1], got.Points[i]
		// Where a day is out of reach the pace is the whole material at once,
		// which is the same pace at every such day, and what separates their
		// costs is how many days of review each is averaged over.
		if now.Short == 0 && was.Short == 0 && now.Minutes > was.Minutes {
			t.Errorf("%s wants %v minutes a day and %s, a day earlier, wants %v",
				got.Days[i], now.Minutes, got.Days[i-1], was.Minutes)
		}
		if now.Share < was.Share-1e-9 {
			t.Errorf("%s gets through %v and %s, a day earlier, gets through %v",
				got.Days[i], now.Share, got.Days[i-1], was.Share)
		}
	}
	inRange(t, got, "now", "suggested")
}

// More days to do the same material in is never more minutes a day: the date
// paces the material, so a nearer day is a dearer one.
//
// It is the pace that thins, so it is said of the days a pace can reach. A day
// out of reach is handed the whole material at once whatever day it is.
func TestTheMinutesOfADateFallAsTheDaysGrow(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		// A review card sent away for three weeks, so a day of the range pays for
		// the reviews that get it there and not for the one that begins it.
		Rule: review.RuleInterval, Interval: 21, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}

	for i, one := range got.Points {
		if one.Minutes <= 0 {
			t.Errorf("%s wants %v minutes a day, and no day is got through for nothing",
				got.Days[i], one.Minutes)
		}
		if i > 0 && one.Short == 0 && got.Points[i-1].Short == 0 &&
			one.Minutes > got.Points[i-1].Minutes {
			t.Errorf("%s wants %v minutes a day and %s, a day earlier, wants %v",
				got.Days[i], one.Minutes, got.Days[i-1], got.Points[i-1].Minutes)
		}
	}
	// And the range holds both regimes, so the fall is said of a pace that
	// thins and not of one that never changed.
	if got.Points[0].Short == 0 || got.Points[len(got.Points)-1].Short != 0 {
		t.Errorf("the range leaves %d review cards short at one end and %d at the other",
			got.Points[0].Short, got.Points[len(got.Points)-1].Short)
	}
	if got.Points[0].Minutes <= got.Points[len(got.Points)-1].Minutes {
		t.Errorf("the first day wants %v minutes a day and the last wants %v",
			got.Points[0].Minutes, got.Points[len(got.Points)-1].Minutes)
	}
}

// A date paces what it holds, so no day of the range is out of reach and the
// material is through by the day the preset aims at. The card counts learned
// beside the date take no part in it.
func TestADateIsMetAtWhateverItCosts(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 25).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 1, ReviewsADay: 45,
		// A review card sent away for three weeks, which twenty-five days leave
		// room for and the days near the left of the range do not.
		Rule: review.RuleInterval, Interval: 21, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.Points {
		if !one.IsEnough {
			t.Errorf("%s is out of reach at %+v, and a date paces what it holds",
				got.Days[i], one)
		}
	}
	if got.Now.Index < 0 {
		t.Fatalf("the preset stands nowhere on its own curve: %+v", got.Now)
	}
	currentPoint := got.Points[got.Now.Index]
	if currentPoint.Share < 1 || !currentPoint.IsEnough || currentPoint.Short != 0 {
		t.Errorf("currentPoint: got Share=%v, IsEnough=%v, Short=%d; want Share>=1, IsEnough=true, Short=0",
			currentPoint.Share, currentPoint.IsEnough, currentPoint.Short)
	}
	if got.Suggested == flashcards.OutsideGrid {
		t.Error("no day is suggested, and the day it aims at is through the material")
	}
}

// A day that has passed is a preset scheduling nothing, and there is no curve
// over it.
func TestADayThatHasPassedHasNoCurve(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 4)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, -3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != 0 || got.Now != flashcards.OutsideGrid || got.Suggested != flashcards.OutsideGrid {
		t.Errorf("curve = %+v, want nothing", got)
	}
}

// The load each day of the week carries and an even load are projected, so what
// stands where the preset stands is worked out with them applied.
func TestTheCurveIsWorkedOutWithTheLoadAndAnEvenLoad(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalRetention, Retention: 0.9,
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	flat, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	light, even := p, p
	light.Load = map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 0}
	even.IsEvenLoad = true

	for name, one := range map[string]review.Preset{"a light week": light, "an even load": even} {
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one)
		if err != nil {
			t.Fatal(err)
		}
		inRange(t, got, "now")
		if got.Now.Index != flat.Now.Index {
			t.Fatalf("%s: got.Now.Index = %d, want %d",
				name, got.Now.Index, flat.Now.Index)
		}
		one, was := got.Points[got.Now.Index], flat.Points[flat.Now.Index]
		if one.Reviews == was.Reviews && one.Minutes == was.Minutes &&
			one.Retained == was.Retained {
			t.Errorf("%s: got point %+v, want different from flat %+v",
				name, one, was)
		}
	}
}

// A curve is over the cards of the decks pointing at this preset, and no
// others. The vault's other deck is another preset's business.
func TestACurveIsOverTheDecksPointingAtThePreset(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	mine, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	rest, err := s.curves(noon).Execute(t.Context(), s.vault, "", p)
	if err != nil {
		t.Fatal(err)
	}

	// Thirty cards of a one-template card model against one, so the day the load
	// wants is not the same day.
	if mine.Points[0].Reviews <= rest.Points[0].Reviews {
		t.Errorf("the preset's deck answers %v a day and the deck naming none answers %v",
			mine.Points[0].Reviews, rest.Points[0].Reviews)
	}
}

// A curve carries how many review cards stand under the preset, so a preset whose
// decks hold nothing is told apart from one nothing points at.
func TestACurveCarriesTheReviewCardsBelowThePreset(t *testing.T) {
	t.Parallel()
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	s := newAnsweredVault(t, 30)
	full, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if full.Decks != 1 || full.Cards != 30 {
		t.Errorf("got Decks = %d, Cards = %d; want Decks = 1, Cards = 30",
			full.Decks, full.Cards)
	}

	empty := openVault(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/Roots.md": deckOf("Sanskrit", 0, 0),
	})
	bare, err := empty.curves(noon).Execute(t.Context(), empty.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if bare.Decks != 1 || bare.Cards != 0 {
		t.Errorf("got Decks = %d, Cards = %d; want Decks = 1, Cards = 0",
			bare.Decks, bare.Cards)
	}
}

// Nothing about a curve is written to the vault: it is shown beside a control,
// and what the control settles is written by the person moving it.
func TestWorkingOutACurveWritesNothingToTheVault(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 4)
	before := read(t, s.vault, "Sanskrit.md")
	deck := read(t, s.vault, "decks/Roots.md")
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	if _, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p); err != nil {
		t.Fatal(err)
	}
	if now := read(t, s.vault, "Sanskrit.md"); now != before {
		t.Errorf("the preset was written\n was %q\n now %q", before, now)
	}
	if now := read(t, s.vault, "decks/Roots.md"); now != deck {
		t.Error("the deck was written")
	}
}

// inRange verifies that points fall within valid grid indices.
func inRange(t *testing.T, c flashcards.Curve, names ...string) {
	t.Helper()
	marks := map[string]flashcards.CurvePoint{"now": c.Now, "suggested": c.Suggested}
	for _, name := range names {
		if marks[name] == flashcards.OutsideGrid {
			t.Errorf("point %s: got OutsideGrid, want valid index in grid of len %d", name, len(c.Grid))
		}
	}
	for name, mark := range marks {
		if mark == flashcards.OutsideGrid {
			continue
		}
		if mark.Index < 0 || mark.Index >= len(c.Grid) {
			t.Errorf("the %s mark is at place %d of %d", name, mark.Index, len(c.Grid))
			continue
		}
		// A mark relies on a place of the grid, so the figures a person reads
		// under it are the figures of the setting they are learned at.
		if mark.Value != c.Grid[mark.Index] {
			t.Errorf("the %s mark is at %v and the place it names is %v",
				name, mark.Value, c.Grid[mark.Index])
		}
		if c.Goal == review.GoalDate && !strings.Contains(mark.Day, "-") {
			t.Errorf("the %s mark names no day: %q", name, mark.Day)
		}
	}
}

// The range a date is chosen from relies on the vault and not on the date, so a
// day the file names close at hand is a range a person can still drag out.
func TestANearDateStillPreservesRoomToGiveYourselfLonger(t *testing.T) {
	t.Parallel()
	s := openVault(t, newStudiedNotes(30))
	// The day after tomorrow, which is the shortest range a person could have
	// left themselves with.
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 2).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	// Thirty unreviewed review cards result in at least a 30-day range.
	if last := got.Grid[len(got.Grid)-1]; last < 30 {
		t.Errorf("a date two days off is a range of %v days, and the material is 30 cards", last)
	}
	if got.Now.Index < 0 {
		t.Errorf("got.Now.Index = %d, want >= 0 (point: %+v)", got.Now.Index, got.Now)
	}
	if got.Points[got.Now.Index].Minutes <= got.Points[len(got.Points)-1].Minutes {
		t.Errorf("the day named costs %v minutes a day and the furthest day %v",
			got.Points[got.Now.Index].Minutes, got.Points[len(got.Points)-1].Minutes)
	}
}

// What a curve says about a backlog is measured over the same horizon on every
// goal.
//
// A goal of a date runs each place to its own day, and a place near the left of
// the range is still asked how long a backlog takes to clear over the horizon
// every other place answers over.
func TestABacklogIsMeasuredOverTheSameHorizonOnEveryGoal(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Grid[0] != 1 {
		t.Fatalf("the curve begins %v days off", got.Grid[0])
	}
	for i, one := range got.Points {
		if len(one.Backlog) < flashcards.Ahead {
			t.Errorf("%v days off carries %d days of backlog, and a clearing is asked "+
				"over %d", got.Grid[i], len(one.Backlog), flashcards.Ahead)
		}
	}
}

// A day at none of the load is no session, so the curve draws the day after it.
//
// A person who has just given today's weekday nothing is the person most in
// need of the picture, and a picture reading nought at every place of the range
// says nothing about the setting it is there to steer.
func TestACurveOnADayAtNoneOfTheLoadDrawsTheNextSession(t *testing.T) {
	t.Parallel()
	s := openVault(t, newStudiedNotes(30))
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	if noon.Weekday() != time.Monday {
		t.Fatalf("the day these curves are drawn on is a %v", noon.Weekday())
	}
	p.Load = map[time.Weekday]int{time.Monday: 0}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Points[0].Reviews == 0 {
		t.Error("the shortest day of the range draws nothing")
	}
	if first, last := got.Points[0].Reviews, got.Points[len(got.Points)-1].Reviews; !(last > first) {
		t.Errorf("the range runs from %g cards to %g, and says nothing about the setting",
			first, last)
	}
}

// A day at none of the load that is not today leaves the curve where it was.
// What it changes is what those days do further out, on the band beside it.
func TestADayAtNoneOfTheLoadAwayFromTodayPreservesTheCurve(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	if noon.Weekday() == time.Saturday {
		t.Fatal("the day these curves are drawn on is the day being given nothing")
	}

	was, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	light := p
	light.Load = map[time.Weekday]int{time.Saturday: 0}
	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", light)
	if err != nil {
		t.Fatal(err)
	}

	for i := range got.Points {
		if got.Points[i].Reviews != was.Points[i].Reviews || got.Points[i].Minutes != was.Points[i].Minutes {
			t.Errorf("at %g the curve draws %g cards in %g minutes, and drew %g in %g",
				got.Grid[i], got.Points[i].Reviews, got.Points[i].Minutes,
				was.Points[i].Reviews, was.Points[i].Minutes)
		}
	}
}

// The count of review cards that cannot be learned in time is reported on the curve.
func TestACurveOfADateReturnsWhatNoPaceReaches(t *testing.T) {
	t.Parallel()
	s := openVault(t, newStudiedNotes(30))
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleInterval, Interval: 21,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cards == 0 {
		t.Fatal("the preset schedules nothing, and there is nothing to fall short")
	}
	if first := got.Points[0]; first.Short != got.Cards {
		t.Errorf("%s leaves %d of the %d review cards short, and a day is no time to learn any",
			got.Days[0], first.Short, got.Cards)
	}
	if last := got.Points[len(got.Points)-1]; last.Short != 0 {
		t.Errorf("%s leaves %d review cards short, and it is %v days off",
			got.Days[len(got.Days)-1], last.Short, got.Grid[len(got.Grid)-1])
	}
	for i := 1; i < len(got.Points); i++ {
		if got.Points[i].Short > got.Points[i-1].Short {
			t.Errorf("%s leaves %d short and %s, a day earlier, leaves %d",
				got.Days[i], got.Points[i].Short, got.Days[i-1], got.Points[i-1].Short)
		}
	}
	// A day nothing can be learned by is met by the pace all the same: every
	// review card that can be learned by it is, and there are none.
	if !got.Points[0].IsEnough {
		t.Error("a day no review card can be learned by is not met by the pace that reaches every one that can")
	}
}

// getLearned is what a curve of these settings says stands learned, at every
// place of it, without repeating a value.
func getLearned(t *testing.T, s vaulted, p review.Preset) []int {
	t.Helper()
	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, one := range got.Points {
		if len(out) == 0 || out[len(out)-1] != one.Learned {
			out = append(out, one.Learned)
		}
	}
	return out
}

// What stands learned today moves with the rule and its threshold, and with
// nothing else.
//
// It is read off the schedules the vault already holds, before a projected day
// is spent, so no budget, no share of the day and no share of the week can
// reach it. What a day's budget is spent on cannot reach it either.
func TestWhatExistsLearnedTodayMovesWithTheRuleAlone(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	// Every card answered on days going back, and nothing asked for months: the
	// intervals are long and the chance of recalling them today is not.
	for back := 240; back >= 180; back -= 15 {
		record := s.run(t, noon.AddDate(0, 0, -back))
		for i := range 30 {
			answer(t, record, cardID(i), 6*time.Second)
		}
	}

	base := review.Defaults()
	base.Goal, base.MinutesADay = review.GoalMinutes, 20
	base.NewADay, base.ReviewsADay = 8, 45
	base.Rule, base.Interval = review.RuleInterval, 21

	learned := getLearned(t, s, base)
	if len(learned) != 1 || learned[0] == 0 {
		t.Fatalf("got learned = %v, want non-empty and non-zero", learned)
	}

	for _, one := range []struct {
		what  string
		alter func(p *review.Preset)
	}{
		{"a budget spent on every showing", func(p *review.Preset) {
			p.Counts = review.BudgetUnitShows
		}},
		{"a day spent on new cards first", func(p *review.Preset) { p.Backlog = 0 }},
		{"an even load off", func(p *review.Preset) { p.IsEvenLoad = false }},
		{"a Saturday at nothing", func(p *review.Preset) {
			p.Load = map[time.Weekday]int{time.Saturday: 0}
		}},
		{"no new cards a day", func(p *review.Preset) { p.NewADay = 0 }},
		{"one review a day", func(p *review.Preset) { p.ReviewsADay = 1 }},
		{"a minute a day", func(p *review.Preset) { p.MinutesADay = 1 }},
		{"a goal of retention", func(p *review.Preset) { p.Goal = review.GoalRetention }},
	} {
		p := base
		one.alter(&p)
		if got := getLearned(t, s, p); !reflect.DeepEqual(got, learned) {
			t.Errorf("%s moved what stands learned from %v to %v", one.what, learned, got)
		}
	}

	// And what does move it: the rule, and the threshold that rule reads.
	other := base
	other.Rule = review.RuleRetention
	if got := getLearned(t, s, other); reflect.DeepEqual(got, learned) {
		t.Errorf("a material months past its answers stands %v learned under both rules", got)
	}
	tighter := base
	tighter.Interval = int(review.IntervalBounds.Max)
	if got := getLearned(t, s, tighter); got[0] >= learned[0] {
		t.Errorf("an interval of a year learns %v of what an interval of three weeks learns %v",
			got, learned)
	}
}

// The day the whole material stands learned is drawn where a preset is steered
// by its minutes and where it is steered by its retention, and nowhere under a
// date: a preset aiming at a day is answered by that day.
//
// It is a prediction under a stated assumption — that every card asked comes
// back — so it is read off a run of its own and not off the run beside it.
func TestTheDayTheMaterialIsLearnedIsGeneratedBelowMinutesAndRetention(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	for back := 60; back >= 15; back -= 15 {
		record := s.run(t, noon.AddDate(0, 0, -back))
		for i := range 20 {
			answer(t, record, cardID(i), 6*time.Second)
		}
	}

	p := review.Defaults()
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 4, 60
	p.Rule, p.Interval = review.RuleInterval, 21

	for _, goal := range []review.Goal{review.GoalMinutes, review.GoalRetention} {
		one := p
		one.Goal = goal
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one)
		if err != nil {
			t.Fatal(err)
		}
		named := false
		for i, place := range got.Points {
			if place.Learns == review.LearnsUnasked {
				t.Errorf("steered by its %s, %v names no day the material is learned",
					goal, got.Grid[i])
			}
			if place.Learns >= 0 {
				named = true
			}
		}
		if !named {
			t.Errorf("steered by its %s, no place of the curve reaches that day", goal)
		}
	}

	dated := p
	dated.Goal = review.GoalDate
	dated.By = noon.AddDate(0, 0, 40).Truncate(24 * time.Hour)
	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", dated)
	if err != nil {
		t.Fatal(err)
	}
	for i, place := range got.Points {
		if place.Learns != review.LearnsUnasked {
			t.Errorf("aiming at a day, %v names day %d as well", got.Grid[i], place.Learns)
		}
	}
}

// The day the file names is a place of the grid, and the mark relies on it.
//
// The picture puts the mark on the range and the person reads the numbers
// beneath it, so those numbers are the day they named and not the day beside
// it.
func TestTheMarkOfADateExistsOnTheDayTheFileNames(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	for _, days := range []int{1, 2, 7, 14, 21, 30, 90, 365} {
		p := review.Preset{
			Goal: review.GoalDate, By: noon.AddDate(0, 0, days).Truncate(24 * time.Hour),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Rule: review.RuleRetention, Retention: 0.9,
		}

		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.Index < 0 {
			t.Fatalf("a date %d days off stands nowhere on its own range", days)
		}
		if got.Days[got.Now.Index] != got.Now.Day {
			t.Errorf("a date %d days off is marked at %s and the place under the mark is %s",
				days, got.Now.Day, got.Days[got.Now.Index])
		}
		if got.Grid[got.Now.Index] != got.Now.Value {
			t.Errorf("a date %d days off is marked at %v days and the place under the mark "+
				"is %v days", days, got.Now.Value, got.Grid[got.Now.Index])
		}
		// The range still begins tomorrow and still runs past the day named.
		if got.Grid[0] != 1 {
			t.Errorf("a date %d days off draws a range beginning %v days off", days, got.Grid[0])
		}
		for i := 1; i < len(got.Grid); i++ {
			if got.Grid[i] <= got.Grid[i-1] {
				t.Fatalf("a date %d days off runs %v then %v",
					days, got.Grid[i-1], got.Grid[i])
			}
		}
	}
}

// Each place of a date's range is read on the day it names.
//
// Past that day the preset schedules nothing, so a debt read off the end of the
// horizon is a debt nobody was asked to pay and a share left in the head is a
// material nobody was asked about.
func TestAPlaceOfADateIsReadOnTheDayItNames(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 30).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Now.Index < 0 {
		t.Fatal("got.Now.Index < 0, want valid index on curve")
	}
	currentPoint := got.Points[got.Now.Index]
	if currentPoint.Share < 1 {
		t.Fatalf("got currentPoint.Share = %v, want >= 1", currentPoint.Share)
	}
	if currentPoint.Owed != 0 {
		t.Errorf("got currentPoint.Owed = %d, want 0", currentPoint.Owed)
	}
	if currentPoint.Retained < 0.5 {
		t.Errorf("got currentPoint.Retained = %v, want >= 0.5", currentPoint.Retained)
	}
}

// A place of a date's range is one run, and everything read off it agrees.
//
// The height of the curve and the mark on it answered out of two runs: the
// height came from the preset's own date, which schedules nothing past the day
// it names, so every place to the right of that day drew a material falling out
// of the head while the mark beside it said the day was met.
func TestAPlaceOfADateIsOneRun(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 5).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.Points {
		if i > 0 && got.Points[i-1].IsEnough && !one.IsEnough {
			t.Errorf("%s is got through and %s, a day later, is not",
				got.Days[i-1], got.Days[i])
		}
	}
}

// A day at none of the load is no session under a date either, so the curve
// draws the day after it.
func TestACurveOfADateOnADayAtNoneOfTheLoadDrawsTheNextSession(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	if noon.Weekday() != time.Monday {
		t.Fatalf("the day these curves are drawn on is a %v", noon.Weekday())
	}
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleRetention, Retention: 0.9,
		Load: map[time.Weekday]int{time.Monday: 0},
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.Points {
		if one.Reviews == 0 {
			t.Fatalf("%s draws nothing for the day of review it names", got.Days[i])
		}
	}
}

// The day suggested for a date is a day the material can be learned by.
//
// It took the soonest day whose cost fitted the minutes the preset keeps and
// asked nothing else, so a range whose first day leaves the whole material out
// of reach was answered with tomorrow.
func TestTheDaySuggestedForADateGetsThroughTheMaterial(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(0, 0, 25).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleInterval, Interval: 21, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Suggested == flashcards.OutsideGrid {
		t.Fatal("got.Suggested == OutsideGrid, want valid suggested point")
	}
	suggestedPoint := got.Points[got.Suggested.Index]
	if suggestedPoint.Short != 0 || !suggestedPoint.IsEnough || suggestedPoint.Share < 1 {
		t.Errorf("%s: got Short = %d, IsEnough = %v, Share = %v; want Short = 0, IsEnough = true, Share >= 1",
			got.Suggested.Day, suggestedPoint.Short, suggestedPoint.IsEnough, suggestedPoint.Share)
	}
	// And it is the soonest such day.
	for i, one := range got.Points {
		if i < got.Suggested.Index && one.Short == 0 && one.IsEnough &&
			one.Minutes <= float64(p.MinutesADay) {
			t.Errorf("%s is suggested and %s, earlier, is through the material too",
				got.Suggested.Day, got.Days[i])
		}
	}
}

// A preset aiming at a day and naming none schedules nothing.
//
// The budget its goal names is the day, and a goal that cannot read its own
// budget is the pause a budget of zero is. Every overdue review card was handed
// over instead, with no count and no minutes to close the day.
func TestADateNamingNoDaySchedulesNothing(t *testing.T) {
	t.Parallel()
	p := review.Preset{
		Goal: review.GoalDate, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}
	if !p.IsPaused(today, noon) {
		t.Error("a preset aiming at a day and naming none schedules something")
	}
	admits := p.GetAllowance(today, noon, review.Spent{}, 30, 0)
	if !admits.IsPaused() {
		t.Errorf("the day of a preset aiming at no day admits %+v", admits)
	}
}

// A day further off than the projection reaches stands nowhere on the range,
// and the range is drawn as far as it goes.
//
// A person who has given themselves a decade sees where their day fell off the
// picture. An empty picture is what a day already past says, and the two are
// not one answer.
func TestADateFurtherOffThanTheProjectionReachesStillDrawsARange(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 2)
	p := review.Preset{
		Goal: review.GoalDate, By: noon.AddDate(20, 0, 0).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: review.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) == 0 {
		t.Fatal("a day twenty years off draws no range at all, as a day already past does")
	}
	if got.Now != flashcards.OutsideGrid {
		t.Errorf("got.Now = %+v, want %+v", got.Now, flashcards.OutsideGrid)
	}
}

// The curve opens on the day a person is already partway through.
//
// The first place of a run is a real day and not a fresh one: what it draws is
// what a session opened now would hand over. An evening re-opening of the tab
// otherwise draws a day already spent as a day still to come.
func TestACurveDrawsWhatIsLeftOfTheDay(t *testing.T) {
	t.Parallel()
	s := openVault(t, newStudiedNotes(30))
	// The session and curve are evaluated for one day.
	write(t, s, "Sanskrit.md", "---\ntype: preset\ngoal: minutes_a_day\n"+
		"minutes_a_day: 3\nnew_a_day: 0\nreviews_a_day: 0\n---\n\n# Sanskrit\n")
	// Every review card owed today, at six seconds an answer.
	before := s.run(t, noon.AddDate(0, 0, -30))
	for i := range 30 {
		answer(t, before, cardID(i), 6*time.Second)
	}
	p := review.Preset{
		Goal: review.GoalMinutes, MinutesADay: 3, NewADay: 0, ReviewsADay: 0,
	}

	fresh, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	// Ten of them answered this morning, which is a minute of the three.
	morning := s.run(t, noon)
	for i := range 10 {
		answer(t, morning, cardID(i), 6*time.Second)
	}
	after, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}

	if after.Points[after.Now.Index].Reviews >= fresh.Points[fresh.Now.Index].Reviews {
		t.Errorf("a day with a minute of it spent draws %v cards and an unspent day %v",
			after.Points[after.Now.Index].Reviews, fresh.Points[fresh.Now.Index].Reviews)
	}
	sat := s.openSessionByPreset(t, today, noon, "Sanskrit.md")
	if got := float64(len(sat.Queue)); got != after.Points[after.Now.Index].Reviews {
		t.Errorf("the session offers %v review cards and the curve draws %v",
			got, after.Points[after.Now.Index].Reviews)
	}
}

// The setting a person is learned at is a place of the grid, so the figures
// under the mark are the figures of that setting.
//
// The mark carried the preset's own value while the figures beside it were the
// grid's nearest place, so a day of fifteen minutes was drawn as a day of
// fourteen and a day of one minute as a day of two.
func TestTheMarkExistsOnTheSettingThePresetContains(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	before := s.run(t, noon.AddDate(0, 0, -30))
	for i := range 30 {
		answer(t, before, cardID(i), 6*time.Second)
	}

	for _, minutes := range []int{1, 3, 15, 20, 200} {
		p := review.Preset{
			Goal: review.GoalMinutes, MinutesADay: minutes, NewADay: 8, ReviewsADay: 45,
		}
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.Index < 0 {
			t.Errorf("minutes %d: got.Now.Index = %d, want >= 0", minutes, got.Now.Index)
			continue
		}
		if got.Grid[got.Now.Index] != float64(minutes) {
			t.Errorf("a day of %d minutes is marked at the place drawing %v minutes",
				minutes, got.Grid[got.Now.Index])
		}
	}

	for _, share := range []float64{0.71, 0.87, 0.9, 0.98} {
		p := review.Preset{
			Goal: review.GoalRetention, Retention: share,
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.Index < 0 || got.Grid[got.Now.Index] != share {
			t.Errorf("a target of %v is marked at place %d, which draws %v",
				share, got.Now.Index, got.Grid[got.Now.Index])
		}
	}
}

// The curve reports why settings schedule no cards.
func TestACurveCarriesTheVerdictOnTheSettingsItWasGeneratedBelow(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	for _, one := range []struct {
		why review.StopReason
		p   review.Preset
	}{
		{review.StoppedNothing, review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
		{review.StoppedNoMinutes, review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 0, NewADay: 8, ReviewsADay: 45,
		}},
		{review.StoppedNoCards, review.Preset{
			Goal: review.GoalRetention, Retention: 0.9,
		}},
		{review.StoppedNoDay, review.Preset{Goal: review.GoalDate}},
		{review.StoppedPastDay, review.Preset{
			Goal: review.GoalDate, By: noon.AddDate(0, 0, -3).Truncate(24 * time.Hour),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
	} {
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one.p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Stops != one.why {
			t.Errorf("%+v draws a curve saying %q, and it schedules nothing for %q",
				one.p, got.Stops, one.why)
		}
	}
}

// A goal keeping no account of what its pace gets through leaves no place of
// its curve falling short.
//
// The stretch a budget does not get through is drawn as a line of its own over
// the curve. A goal of minutes and a goal of retention set no pace at a day,
// and every place of theirs stands as one nothing is short of.
func TestOnlyADateDrawsAPlaceAsFallingShort(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	for _, goal := range []review.Goal{review.GoalMinutes, review.GoalRetention} {
		p := review.Preset{
			Goal: goal, MinutesADay: 20, NewADay: 8, ReviewsADay: 45, Retention: 0.9,
		}

		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		for i, one := range got.Points {
			if !one.IsEnough {
				t.Errorf("%s draws place %d as one the budget does not get through", goal, i)
				break
			}
		}
	}
}

// A curve is drawn under the settings a save would take.
//
// A control asks for a curve at the value under the hand, and that value is
// written the moment the hand is release. Settings the writer refuses draw no
// curve.
func TestACurveIsRefusedTheSettingsASaveIsRefused(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 4)
	for _, one := range []struct {
		named  string
		preset review.Preset
	}{
		{"a chance of recall past one", review.Preset{
			Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 45, Retention: 5,
		}},
		{"a chance of recall at nothing", review.Preset{
			Goal: review.GoalRetention, NewADay: 8, ReviewsADay: 45,
		}},
		{"a day longer than one runs", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 1 << 30, NewADay: 8, ReviewsADay: 45,
		}},
	} {
		_, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one.preset)
		if !errors.Is(err, flashcards.ErrOutOfBounds) {
			t.Errorf("%s draws a curve, and answered %v", one.named, err)
		}
	}
}

// A build carrying no index reaches no deck, and answers ErrNotCarried.
func TestABuildWithNoIndexDrawsNoCurve(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 4)
	noIndex := flashcards.NewListReviewCards(s.env.Ports.Readers, nil, nil)
	u := flashcards.NewProjectCurve(noIndex, s.schedules, s.presets, today, func() time.Time { return noon })

	p := review.Preset{
		Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}
	if _, err := u.Execute(t.Context(), s.vault, "Sanskrit.md", p); !errors.Is(err, flashcards.ErrNotCarried) {
		t.Errorf("a build with no index answered %v", err)
	}
}

// A curve works every place of its grid out alongside the others, and the curve
// it evaluates to is the same curve every time it is asked for.
//
// Run under the race detector this is what holds the places to sharing nothing:
// each run reads the schedules it is given and writes only what it returns.
// Read without it, what it holds is that the answer does not depend on the order
// the places happen to finish in, which is the order of the cores the machine
// has.
func TestACurveIsTheSameCurveEveryTimeItIsAsked(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 200)
	curves := s.curves(noon)
	for _, one := range atOnceGoals() {
		t.Run(one.name, func(t *testing.T) {
			first, err := curves.Execute(t.Context(), s.vault, "Sanskrit.md", one.preset)
			if err != nil {
				t.Fatal(err)
			}
			want := describeCurve(first)
			for again := range 8 {
				got, err := curves.Execute(t.Context(), s.vault, "Sanskrit.md", one.preset)
				if err != nil {
					t.Fatal(err)
				}
				if describeCurve(got) != want {
					t.Fatalf("asking %d times over gives\n%s\nand the first asking gave\n%s",
						again+2, describeCurve(got), want)
				}
			}
		})
	}
}

// atOnceGoals is one preset under each goal, so that all three of the loops
// that work a grid out are asked the question.
func atOnceGoals() []struct {
	name   string
	preset review.Preset
} {
	return []struct {
		name   string
		preset review.Preset
	}{
		{"minutes", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.87, Rule: review.RuleInterval, Interval: 21,
			Counts: review.BudgetUnitCards, Backlog: 100, IsEvenLoad: true,
		}},
		{"retention", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21,
			Counts: review.BudgetUnitCards, Backlog: 60, IsEvenLoad: true,
		}},
		{"a date", review.Preset{
			Goal: review.GoalDate, By: noon.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.BudgetUnitCards, IsEvenLoad: true,
		}},
	}
}

// describeCurve is a whole curve written out, every scalar and every series of
// it, so that two of them are compared by what they say and not by what they
// point at.
func describeCurve(c projection.Curve) string {
	return fmt.Sprintf("%v", c)
}

// curveLoad is the vault the window's requests are measured on, with its decks
// pointing at two presets: one deck points at the first and the other nineteen
// at the second.
func curveLoad(cards int) map[string]string {
	const settings = "goal: minutes_a_day\nminutes_a_day: 20\nnew_a_day: 8\n" +
		"reviews_a_day: 45\nretention: 0.87\n"
	out := loadDeck(cards)
	out["Small.md"] = "---\ntype: preset\n" + settings + "---\n"
	out["Large.md"] = "---\ntype: preset\n" + settings + "---\n"
	for item := range 20 {
		to := "Large"
		if item == 0 {
			to = "Small"
		}
		path := fmt.Sprintf("decks/Deck%02d.md", item)
		out[path] = strings.Replace(out[path], "---\ntype: deck\n",
			"---\ntype: deck\n"+createPresetLink(to), 1)
	}
	return out
}

// curveWhole is the vault of twenty decks with every one of them pointing at
// one preset, so the curve of that preset is drawn over the whole of it.
func curveWhole(cards int) map[string]string {
	out := loadDeck(cards)
	out["Whole.md"] = "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 20\n" +
		"new_a_day: 8\nreviews_a_day: 45\nretention: 0.87\n---\n"
	for item := range 20 {
		path := fmt.Sprintf("decks/Deck%02d.md", item)
		out[path] = strings.Replace(out[path], "---\ntype: deck\n",
			"---\ntype: deck\n"+createPresetLink("Whole"), 1)
	}
	return out
}

// The curve of a preset holding the whole vault, at four sizes of it.
//
// What the projection costs relies on the review cards the preset holds, and
// these four rows are what says how it grows with them. The log grows with the
// vault too, at the same half an answer a review card the row above is built on.
//
// The schedule cache is filled before the clock starts, which is what opening
// the window and then a preset tab costs.
func BenchmarkCurveCards(b *testing.B) {
	for _, cards := range []int{500, 5000, 20000, 50000} {
		b.Run(fmt.Sprint(cards), func(b *testing.B) {
			s := openVault(b, curveWhole(cards))
			loadAnswers(b, s, time.Now(), cards, 180, max(1, cards*150/50000))

			ctx := b.Context()
			if _, err := s.schedules.Execute(ctx, s.vault); err != nil {
				b.Fatal(err)
			}
			p := review.Preset{
				Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			}
			curves := s.curves(time.Now())
			if _, err := curves.Execute(ctx, s.vault, "Whole.md", p); err != nil {
				b.Fatal(err)
			}
			for b.Loop() {
				if _, err := curves.Execute(ctx, s.vault, "Whole.md", p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// The curve of one preset, over fifty thousand review cards in twenty decks
// answered a hundred and fifty times a day for six months.
//
// One preset holds a deck of the vault and the other holds the rest of it, so
// what the two rows are read against each other for is what a curve pays for
// the decks it does not schedule.
//
// The schedule cache is filled before the clock starts, which is what opening
// the window and then a preset tab costs.
func BenchmarkPresetCurve(b *testing.B) {
	const cards = 50000
	s := openVault(b, curveLoad(cards))
	loadAnswers(b, s, time.Now(), cards, 180, 150)

	ctx := b.Context()
	if _, err := s.schedules.Execute(ctx, s.vault); err != nil {
		b.Fatal(err)
	}
	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	curves := s.curves(time.Now())

	for _, path := range []string{"Small.md", "Large.md"} {
		if _, err := curves.Execute(ctx, s.vault, path, p); err != nil {
			b.Fatal(err)
		}
		b.Run(strings.TrimSuffix(path, ".md"), func(b *testing.B) {
			for b.Loop() {
				if _, err := curves.Execute(ctx, s.vault, path, p); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// closing is the vault's files with the request cancelled at the first one
// read, which is a person shutting the window while its decks are being read.
type closing struct {
	inner  ports.VaultReaders
	cancel context.CancelFunc
}

func (r closing) Open(v domain.Vault) (ports.VaultReader, error) {
	one, err := r.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return closingReader{VaultReader: one, cancel: r.cancel}, nil
}

type closingReader struct {
	ports.VaultReader
	cancel context.CancelFunc
}

func (r closingReader) Read(ctx context.Context, path string) ([]byte, error) {
	r.cancel()
	return r.VaultReader.Read(ctx, path)
}

// A deck that could not be read contributes no review card, and a request that
// was cancelled read none of them. A curve over no cards is not the answer to
// what the preset holds.
func TestACurveRefusesARequestThatIsMissing(t *testing.T) {
	t.Parallel()
	s := newAnsweredVault(t, 30)
	ctx, cancel := context.WithCancel(t.Context())

	customCards := flashcards.NewListReviewCards(closing{inner: s.env.Ports.Readers, cancel: cancel}, s.notes, s.links)
	curves := flashcards.NewProjectCurve(customCards, s.schedules, s.presets, today, func() time.Time { return noon })

	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	got, err := curves.Execute(ctx, s.vault, "Sanskrit.md", p)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a curve drawn after the request was cancelled gave %+v, %v", got, err)
	}
}

// sanskrit is the identifier the preset of that name carries, so that a deck
// can name it the auxiliary way.
const sanskrit = "01M02ACGM0FYMSXNDP29C90JNR"

// newPointedNotes is a vault of three presets and five decks. Three decks name
// one preset — one of them by its identifier and two by its name — one deck
// names another preset, one names none, and the third preset is named by
// nothing.
func newPointedNotes() map[string]string {
	return map[string]string{
		"Term.md": vault["Term.md"],
		"Sanskrit.md": "---\ntype: preset\nid: " + sanskrit + "\ngoal: minutes_a_day\n" +
			"minutes_a_day: 20\nnew_a_day: 8\nreviews_a_day: 45\nretention: 0.87\n---\n\n# Sanskrit\n",
		"Grammar.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 15\n" +
			"new_a_day: 5\nreviews_a_day: 30\nretention: 0.9\n---\n\n# Grammar\n",
		"Quiet.md": "---\ntype: preset\ngoal: minutes_a_day\nminutes_a_day: 10\n" +
			"new_a_day: 4\nreviews_a_day: 20\nretention: 0.85\n---\n\n# Quiet\n",
		"decks/Roots.md":  newPointingDeck(createPresetLink("Sanskrit"), 0, 8),
		"decks/Verbs.md":  newPointingDeck(createPresetLink("Sanskrit"), 8, 8),
		"decks/Chants.md": newPointingDeck(createPresetLink("note://"+sanskrit), 16, 8),
		"decks/Cases.md":  newPointingDeck(createPresetLink("Grammar"), 24, 8),
		"decks/Other.md":  newPointingDeck("", 32, 8),
	}
}

// createPresetLink is the `links:` block of a deck naming its preset.
func createPresetLink(to string) string {
	return "links:\n  - to: " + to + "\n    role: ref\n    type: preset\n"
}

// newPointingDeck is a deck of cards cards, the first of them numbered from.
func newPointingDeck(links string, from, cards int) string {
	out := "---\ntype: deck\n" + links + "---\n"
	for i := from; i < from+cards; i++ {
		out += fmt.Sprintf(
			"\n## Word %d ^%s\n\n[[Term]]\n\n### Word\n\nWord %d\n\n### Meaning\n\nMeaning %d\n",
			i, cardID(i), i, i)
	}
	return out
}

// recordHistoryAnswers writes the same history into a vault: two cards of every deck
// answered three times each in the months before the day the curves are drawn
// on, with the times answers have — a middle of a few seconds and a tail of
// interruptions.
//
// Two vaults given this hold one history, so a curve over either is drawn from
// the same answers.
func recordHistoryAnswers(t *testing.T, s vaulted) {
	t.Helper()
	for _, days := range []int{150, 120, 90} {
		record := s.run(t, noon.AddDate(0, 0, -days))
		for card := 0; card < 40; card += 8 {
			answer(t, record, cardID(card), 4*time.Second)
			answer(t, record, cardID(card+1), 50*time.Second)
		}
	}
}

// The decks a curve is worked out over are the decks pointing at the preset,
// and how many of them there are is counted the same way.
func countScheduled(t *testing.T, s vaulted, path string) (decks []string, reviewCards int) {
	t.Helper()
	target, err := s.standings.GetDeckPaths(t.Context(), s.vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, deck := range target {
		p, err := s.presets.GetForDeck(t.Context(), s.vault, deck)
		if err != nil {
			t.Fatal(err)
		}
		if p.Path == path {
			decks = append(decks, deck)
		}
	}
	return decks, len(s.standings.GetReviewCards(t.Context(), s.vault, decks))
}

// A curve reads the decks pointing at its preset and leaves the rest of the
// vault unread, and what it evaluates to is what reading the whole vault evaluates to:
// the same range, the same value at every place of it, the same marks, and the
// same decks and review cards behind them.
//
// The vault it is asked of holds decks of other presets and decks of none, and
// the one it is read against holds only the decks the preset schedules. Both
// hold one history, so what the two answers differ by is the reading.
func TestACurveIsGeneratedFromTheDecksPointingAtThePreset(t *testing.T) {
	t.Parallel()
	shapes := []struct {
		name string
		path string
	}{
		{"a preset several decks point at", "Sanskrit.md"},
		{"a preset one deck points at", "Grammar.md"},
		{"a preset nothing points at", "Quiet.md"},
	}
	goals := []struct {
		name string
		p    review.Preset
	}{
		{"minutes", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
		{"retention", review.Preset{
			Goal: review.GoalRetention, Retention: 0.87,
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}},
		{"a date", review.Preset{
			Goal: review.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Rule: review.RuleInterval, Interval: 21, Retention: 0.9,
		}},
	}

	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			whole := openVault(t, newPointedNotes())
			recordHistoryAnswers(t, whole)
			decks, reviewCards := countScheduled(t, whole, shape.path)

			// The same vault with every deck the preset does not schedule taken
			// out of it, where there is nothing to narrow.
			notes := maps.Clone(newPointedNotes())
			for path := range notes {
				if strings.HasPrefix(path, "decks/") && !slices.Contains(decks, path) {
					delete(notes, path)
				}
			}
			alone := openVault(t, notes)
			recordHistoryAnswers(t, alone)

			for _, goal := range goals {
				t.Run(goal.name, func(t *testing.T) {
					got, err := whole.curves(noon).
						Execute(t.Context(), whole.vault, shape.path, goal.p)
					if err != nil {
						t.Fatal(err)
					}
					want, err := alone.curves(noon).
						Execute(t.Context(), alone.vault, shape.path, goal.p)
					if err != nil {
						t.Fatal(err)
					}

					for _, one := range []projection.Curve{got, want} {
						if one.Decks != len(decks) || one.Cards != reviewCards {
							t.Errorf("the curve carries %d decks and %d review cards, and %d decks "+
								"point at %s with %d review cards in them",
								one.Decks, one.Cards, len(decks), shape.path, reviewCards)
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("over the whole vault:\n%s\nover its decks alone:\n%s",
							curveOf(got), curveOf(want))
					}
				})
			}
		})
	}
}

// curveOf is a curve in one line of a failure.
func curveOf(c projection.Curve) string {
	return fmt.Sprintf("%d decks, %d review cards, %d overdue, %d unbegun\ngrid %v\nnow %+v"+
		"\nsuggested %+v\nat %+v", c.Decks, c.Cards, c.Overdue, c.Unbegun, c.Grid,
		c.Now, c.Suggested, c.Points)
}
