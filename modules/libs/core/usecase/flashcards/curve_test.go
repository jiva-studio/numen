package flashcards_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
)

// studied is a vault of one preset, one deck of thirty cards pointing at it,
// and a second deck pointing nowhere.
func studied(cards int) map[string]string {
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

// curves is the simulator over one vault, on a named day.
func (s vaulted) curves(now time.Time) flashcards.Curves {
	return flashcards.Curves{
		Standings: s.standings, Schedules: s.kept, Presets: s.presets,
		Day: today, Now: func() time.Time { return now },
	}
}

// noon is a fixed hour, so that a curve is the same curve whenever the tests
// are run.
var noon = time.Date(2026, 3, 2, 12, 0, 0, 0, time.Local)

// A curve is asked for once per position of a control a person is dragging, and
// none of those asks writes anything.
func TestACurveWritesNoScheduleCache(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	if _, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p); err != nil {
		t.Fatal(err)
	}
	if raw, err := s.kept.Kept.Read(t.Context(), s.vault.ID); err == nil {
		t.Errorf("a curve wrote a cache of %d bytes", len(raw))
	}
}

// The whole range of the goal is worked out in one pass: a control moving over
// it reads a finished array and computes nothing.
func TestTheCurveOfMinutesCoversTheWholeRange(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != flashcards.Points || len(got.At) != flashcards.Points {
		t.Fatalf("the curve has %d places and %d values", len(got.Grid), len(got.At))
	}
	for i := 1; i < len(got.Grid); i++ {
		if got.Grid[i] <= got.Grid[i-1] {
			t.Fatalf("the grid runs %v then %v", got.Grid[i-1], got.Grid[i])
		}
		if got.At[i].Reviews < got.At[i-1].Reviews {
			t.Errorf("a longer day answers %v, a shorter one %v",
				got.At[i].Reviews, got.At[i-1].Reviews)
		}
		if got.At[i].Owed > got.At[i-1].Owed {
			t.Errorf("a longer day leaves %d owed, a shorter one %d",
				got.At[i].Owed, got.At[i-1].Owed)
		}
	}
	inRange(t, got)
	if got.At[len(got.At)-1].Owed != 0 {
		t.Errorf("the longest day on the grid still leaves %d owed", got.At[len(got.At)-1].Owed)
	}
}

// A higher target is shorter intervals, so the curve of what a day costs rises
// with it. Nothing is suggested: what a target leaves in the head climbs the
// whole way, and a mark on the most of it would stand at the far end every time.
func TestTheCurveOfRetentionCoversTheWholeRange(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalRetention, Retention: 0.87, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != flashcards.Points {
		t.Fatalf("the curve has %d places", len(got.Grid))
	}
	if got.Grid[0] != history.RetentionBounds.Least ||
		got.Grid[len(got.Grid)-1] != history.RetentionBounds.Most {
		t.Errorf("the grid runs from %v to %v", got.Grid[0], got.Grid[len(got.Grid)-1])
	}
	for i := 1; i < len(got.At); i++ {
		if got.At[i].Minutes < got.At[i-1].Minutes-1e-9 {
			t.Errorf("a target of %v costs %v minutes, one of %v costs %v",
				got.Grid[i], got.At[i].Minutes, got.Grid[i-1], got.At[i-1].Minutes)
		}
	}
	inRange(t, got)

	if got.Suggested != flashcards.Nowhere {
		t.Errorf("a target of %v is suggested, and this goal points at none",
			got.Suggested.Value)
	}
	// What the cost buys climbs the range rather than peaking inside it, which
	// is why there is nothing to point at.
	least, most := got.At[0].Retained, got.At[len(got.At)-1].Retained
	if most <= least {
		t.Errorf("the easiest target retains %v and the hardest %v", least, most)
	}
}

// A goal of a date runs from tomorrow to a day further off than the one the
// file names, so the day a person is on stands inside the range and they can
// always give themselves longer. A later day never costs more than an earlier
// one.
func TestTheCurveOfADateRunsPastTheDayNamed(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != len(got.Days) || len(got.Grid) != len(got.At) {
		t.Fatalf("%d places, %d days, %d values", len(got.Grid), len(got.Days), len(got.At))
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
	if got.Now.Value != 20 || got.Now.Day != p.By.Format(history.Named) {
		t.Errorf("the preset stands at %+v, and the day it names is 20 days off", got.Now)
	}
	if got.Now.At <= 0 || got.Now.At >= len(got.Grid)-1 {
		t.Errorf("the day named stands at place %d of %d, and is not inside the range",
			got.Now.At, len(got.Grid))
	}

	for i := 1; i < len(got.At); i++ {
		was, now := got.At[i-1], got.At[i]
		if now.Minutes > was.Minutes {
			t.Errorf("%s wants %v minutes a day and %s, a day earlier, wants %v",
				got.Days[i], now.Minutes, got.Days[i-1], was.Minutes)
		}
		if now.Through < was.Through-1e-9 {
			t.Errorf("%s gets through %v and %s, a day earlier, gets through %v",
				got.Days[i], now.Through, got.Days[i-1], was.Through)
		}
	}
	inRange(t, got)
}

// More days to do the same material in is never more minutes a day: the date
// paces the material, so a nearer day is a dearer one.
func TestTheMinutesOfADateFallAsTheDaysGrow(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}

	for i, one := range got.At {
		if one.Minutes <= 0 {
			t.Errorf("%s wants %v minutes a day, and no day is got through for nothing",
				got.Days[i], one.Minutes)
		}
		if i > 0 && one.Minutes > got.At[i-1].Minutes {
			t.Errorf("%s wants %v minutes a day and %s, a day earlier, wants %v",
				got.Days[i], one.Minutes, got.Days[i-1], got.At[i-1].Minutes)
		}
	}
	if got.At[0].Minutes <= got.At[len(got.At)-1].Minutes {
		t.Errorf("the first day wants %v minutes a day and the last wants %v",
			got.At[0].Minutes, got.At[len(got.At)-1].Minutes)
	}
}

// A date paces what it holds, so no day of the range is out of reach and the
// material is through by the day the preset aims at. The card counts standing
// beside the date take no part in it.
func TestADateIsMetAtWhateverItCosts(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 2).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 1, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.At {
		if !one.Met {
			t.Errorf("%s is out of reach at %+v, and a date paces what it holds",
				got.Days[i], one)
		}
	}
	last := got.At[len(got.At)-1]
	if last.Through < 1 || !last.Enough {
		t.Errorf("the day it aims at gets through %v of the material", last.Through)
	}
	if got.Suggested == flashcards.Nowhere {
		t.Error("no day is suggested, and the day it aims at is through the material")
	}
}

// A day that has passed is a preset scheduling nothing, and there is no curve
// over it.
func TestADayThatHasPassedHasNoCurve(t *testing.T) {
	s := opened(t, studied(4))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, -3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) != 0 || got.Now != flashcards.Nowhere || got.Suggested != flashcards.Nowhere {
		t.Errorf("curve = %+v, want nothing", got)
	}
}

// The load each day of the week carries and an even load are projected, so what
// stands where the preset stands is worked out with them applied.
func TestTheCurveIsWorkedOutWithTheLoadAndAnEvenLoad(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalRetention, Retention: 0.9,
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	flat, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	light, even := p, p
	light.Load = map[time.Weekday]int{time.Wednesday: 50, time.Sunday: 0}
	even.EvenLoad = true

	for name, one := range map[string]history.Preset{"a light week": light, "an even load": even} {
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one)
		if err != nil {
			t.Fatal(err)
		}
		inRange(t, got)
		if got.Now.At != flat.Now.At {
			t.Fatalf("with %s the preset stands at place %d, and without at %d",
				name, got.Now.At, flat.Now.At)
		}
		one, was := got.At[got.Now.At], flat.At[flat.Now.At]
		if one.Reviews == was.Reviews && one.Minutes == was.Minutes &&
			one.Retained == was.Retained {
			t.Errorf("with %s the preset comes to %+v, the same as a preset keeping neither",
				name, one)
		}
	}
}

// A curve is over the cards of the decks pointing at this preset, and no
// others. The vault's other deck is another preset's business.
func TestACurveIsOverTheDecksPointingAtThePreset(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	mine, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	rest, err := s.curves(noon).Execute(t.Context(), s.vault, "", p)
	if err != nil {
		t.Fatal(err)
	}

	// Thirty cards of a one-faced stencil against one, so the day the load
	// wants is not the same day.
	if mine.At[0].Reviews <= rest.At[0].Reviews {
		t.Errorf("the preset's deck answers %v a day and the deck naming none answers %v",
			mine.At[0].Reviews, rest.At[0].Reviews)
	}
}

// A curve carries how many card faces stand under the preset, so a preset whose
// decks hold nothing is told apart from one nothing points at.
func TestACurveCarriesTheCardFacesUnderThePreset(t *testing.T) {
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	s := opened(t, studied(30))
	full, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if full.Decks != 1 || full.Cards != 30 {
		t.Errorf("a deck of thirty cards of a one-faced stencil came to %d decks and %d cards",
			full.Decks, full.Cards)
	}

	empty := opened(t, map[string]string{
		"Term.md":        term,
		"Sanskrit.md":    preset("new_a_day: 8\nreviews_a_day: 45\n"),
		"decks/Roots.md": deckOf("Sanskrit", 0, 0),
	})
	bare, err := empty.curves(noon).Execute(t.Context(), empty.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if bare.Decks != 1 || bare.Cards != 0 {
		t.Errorf("a deck holding no cards came to %d decks and %d cards", bare.Decks, bare.Cards)
	}
}

// Nothing about a curve is written to the vault: it is shown beside a control,
// and what the control settles is written by the person moving it.
func TestWorkingOutACurveWritesNothingToTheVault(t *testing.T) {
	s := opened(t, studied(4))
	before := read(t, s.vault, "Sanskrit.md")
	deck := read(t, s.vault, "decks/Roots.md")
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

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

// inRange says the two marks fall on the curve.
func inRange(t *testing.T, c flashcards.Curve) {
	t.Helper()
	for name, mark := range map[string]flashcards.Mark{"now": c.Now, "suggested": c.Suggested} {
		if mark == flashcards.Nowhere {
			continue
		}
		if mark.At < 0 || mark.At >= len(c.Grid) {
			t.Errorf("the %s mark is at place %d of %d", name, mark.At, len(c.Grid))
			continue
		}
		if mark.Value != c.Grid[mark.At] && name == "suggested" {
			t.Errorf("the suggested mark is at %v and the place it names is %v",
				mark.Value, c.Grid[mark.At])
		}
		if c.Goal == history.GoalDate && !strings.Contains(mark.Day, "-") {
			t.Errorf("the %s mark names no day: %q", name, mark.Day)
		}
	}
}

// The range a date is chosen from stands on the vault and not on the date, so a
// day the file names close at hand is a range a person can still drag out.
func TestANearDateStillLeavesRoomToGiveYourselfLonger(t *testing.T) {
	s := opened(t, studied(30))
	// The day after tomorrow, which is the shortest range a person could have
	// left themselves with.
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 2).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	// Thirty card faces nobody has answered is thirty days at one a day, which
	// is the slowest a day of review goes.
	if last := got.Grid[len(got.Grid)-1]; last < 30 {
		t.Errorf("a date two days off is a range of %v days, and the material is 30 cards", last)
	}
	if got.Now.At < 0 {
		t.Errorf("the day named stands nowhere on the range: %+v", got.Now)
	}
	if got.At[got.Now.At].Minutes <= got.At[len(got.At)-1].Minutes {
		t.Errorf("the day named costs %v minutes a day and the furthest day %v",
			got.At[got.Now.At].Minutes, got.At[len(got.At)-1].Minutes)
	}
}

// What a curve says about a backlog is measured over the same horizon on every
// goal.
//
// A goal of a date runs each place to its own day, and a place near the left of
// the range would otherwise be asked how long a backlog takes to clear over a
// horizon of a day or two and answer that it never does.
func TestABacklogIsMeasuredOverTheSameHorizonOnEveryGoal(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Grid[0] != 1 {
		t.Fatalf("the curve begins %v days off", got.Grid[0])
	}
	for i, one := range got.At {
		if len(one.Backlog) < history.Ahead {
			t.Errorf("%v days off carries %d days of backlog, and a clearing is asked "+
				"over %d", got.Grid[i], len(one.Backlog), history.Ahead)
		}
	}
}

// A day at none of the load is no sitting, so the curve draws the day after it.
//
// A person who has just given today's weekday nothing is the person most in
// need of the picture, and a picture reading nought at every place of the range
// says nothing about the setting it is there to steer.
func TestACurveOnADayAtNoneOfTheLoadDrawsTheNextSitting(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	if noon.Weekday() != time.Monday {
		t.Fatalf("the day these curves are drawn on is a %v", noon.Weekday())
	}
	p.Load = map[time.Weekday]int{time.Monday: 0}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.At[0].Reviews == 0 {
		t.Error("the shortest day of the range draws nothing")
	}
	if first, last := got.At[0].Reviews, got.At[len(got.At)-1].Reviews; !(last > first) {
		t.Errorf("the range runs from %g cards to %g, and says nothing about the setting",
			first, last)
	}
}

// A day at none of the load that is not today leaves the curve where it was.
// What it changes is what those days do further out, on the band beside it.
func TestADayAtNoneOfTheLoadAwayFromTodayLeavesTheCurve(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
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

	for i := range got.At {
		if got.At[i].Reviews != was.At[i].Reviews || got.At[i].Minutes != was.At[i].Minutes {
			t.Errorf("at %g the curve draws %g cards in %g minutes, and drew %g in %g",
				got.Grid[i], got.At[i].Reviews, got.At[i].Minutes,
				was.At[i].Reviews, was.At[i].Minutes)
		}
	}
}
