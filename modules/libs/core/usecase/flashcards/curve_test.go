package flashcards_test

import (
	"fmt"
	"reflect"
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
		// A card face sent away for three weeks, so the days near the left of
		// the range cannot learn one and the days to the right can.
		Rule: history.RuleInterval, Interval: 21, Retention: 0.9,
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
		// Where a day is out of reach the pace is the whole material at once,
		// which is the same pace at every such day, and what separates their
		// costs is how many days of review each is averaged over.
		if now.Short == 0 && was.Short == 0 && now.Minutes > was.Minutes {
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
//
// It is the pace that thins, so it is said of the days a pace can reach. A day
// out of reach is handed the whole material at once whatever day it is.
func TestTheMinutesOfADateFallAsTheDaysGrow(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		// A card face sent away for three weeks, so a day of the range pays for
		// the reviews that get it there and not for the one that begins it.
		Rule: history.RuleInterval, Interval: 21, Retention: 0.9,
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
		if i > 0 && one.Short == 0 && got.At[i-1].Short == 0 &&
			one.Minutes > got.At[i-1].Minutes {
			t.Errorf("%s wants %v minutes a day and %s, a day earlier, wants %v",
				got.Days[i], one.Minutes, got.Days[i-1], got.At[i-1].Minutes)
		}
	}
	// And the range holds both regimes, so the fall is said of a pace that
	// thins and not of one that never changed.
	if got.At[0].Short == 0 || got.At[len(got.At)-1].Short != 0 {
		t.Errorf("the range leaves %d card faces short at one end and %d at the other",
			got.At[0].Short, got.At[len(got.At)-1].Short)
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
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 25).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 1, ReviewsADay: 45,
		// A card face sent away for three weeks, which twenty-five days leave
		// room for and the days near the left of the range do not.
		Rule: history.RuleInterval, Interval: 21, Retention: 0.9,
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
	if got.Now.At < 0 {
		t.Fatalf("the preset stands nowhere on its own curve: %+v", got.Now)
	}
	stands := got.At[got.Now.At]
	if stands.Through < 1 || !stands.Enough || stands.Short != 0 {
		t.Errorf("the day it aims at gets through %v of the material, and %d of it stands short",
			stands.Through, stands.Short)
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
		// A mark stands on a place of the grid, so the figures a person reads
		// under it are the figures of the setting they are standing at.
		if mark.Value != c.Grid[mark.At] {
			t.Errorf("the %s mark is at %v and the place it names is %v",
				name, mark.Value, c.Grid[mark.At])
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

// A date the rule cannot be met by says so on the picture.
//
// The number of card faces no pace reaches stands at each place of the range,
// and falls away as the day moves off. Nothing is hidden by it: the pace beside
// it is the one that reaches every card face that can be reached, and where
// none can be, that is what the number says.
func TestACurveOfADateSaysWhatNoPaceReaches(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 3).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleInterval, Interval: 21,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cards == 0 {
		t.Fatal("the preset schedules nothing, and there is nothing to fall short")
	}
	if first := got.At[0]; first.Short != got.Cards {
		t.Errorf("%s leaves %d of the %d card faces short, and a day is no time to learn any",
			got.Days[0], first.Short, got.Cards)
	}
	if last := got.At[len(got.At)-1]; last.Short != 0 {
		t.Errorf("%s leaves %d card faces short, and it is %v days off",
			got.Days[len(got.Days)-1], last.Short, got.Grid[len(got.Grid)-1])
	}
	for i := 1; i < len(got.At); i++ {
		if got.At[i].Short > got.At[i-1].Short {
			t.Errorf("%s leaves %d short and %s, a day earlier, leaves %d",
				got.Days[i], got.At[i].Short, got.Days[i-1], got.At[i-1].Short)
		}
	}
	// A day nothing can be learned by is met by the pace all the same: every
	// card face that can be learned by it is, and there are none.
	if !got.At[0].Enough {
		t.Error("a day no card face can be learned by is not met by the pace that reaches every one that can")
	}
}

// learnedAt is what a curve of these settings says stands learned, at every
// place of it, without repeating a value.
func learnedAt(t *testing.T, s vaulted, p history.Preset) []int {
	t.Helper()
	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	var out []int
	for _, one := range got.At {
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
func TestWhatStandsLearnedTodayMovesWithTheRuleAlone(t *testing.T) {
	s := opened(t, studied(30))
	// Every card answered on days going back, and nothing asked for months: the
	// intervals are long and the chance of recalling them today is not.
	for back := 240; back >= 180; back -= 15 {
		record := s.run(t, noon.AddDate(0, 0, -back))
		for i := range 30 {
			answer(t, record, mark(i), 6*time.Second)
		}
	}

	base := history.Defaults()
	base.Goal, base.MinutesADay = history.GoalMinutes, 20
	base.NewADay, base.ReviewsADay = 8, 45
	base.Rule, base.Interval = history.RuleInterval, 21

	standing := learnedAt(t, s, base)
	if len(standing) != 1 || standing[0] == 0 {
		t.Fatalf("the vault stands at %v learned, and there is nothing to hold still", standing)
	}

	for _, one := range []struct {
		what  string
		alter func(p *history.Preset)
	}{
		{"a budget spent on every showing", func(p *history.Preset) {
			p.Counts = history.CountsShows
		}},
		{"a day spent on new cards first", func(p *history.Preset) { p.Backlog = 0 }},
		{"an even load off", func(p *history.Preset) { p.EvenLoad = false }},
		{"a Saturday at nothing", func(p *history.Preset) {
			p.Load = map[time.Weekday]int{time.Saturday: 0}
		}},
		{"no new cards a day", func(p *history.Preset) { p.NewADay = 0 }},
		{"one review a day", func(p *history.Preset) { p.ReviewsADay = 1 }},
		{"a minute a day", func(p *history.Preset) { p.MinutesADay = 1 }},
		{"a goal of retention", func(p *history.Preset) { p.Goal = history.GoalRetention }},
	} {
		p := base
		one.alter(&p)
		if got := learnedAt(t, s, p); !reflect.DeepEqual(got, standing) {
			t.Errorf("%s moved what stands learned from %v to %v", one.what, standing, got)
		}
	}

	// And what does move it: the rule, and the threshold that rule reads.
	other := base
	other.Rule = history.RuleRetention
	if got := learnedAt(t, s, other); reflect.DeepEqual(got, standing) {
		t.Errorf("a material months past its answers stands %v learned under both rules", got)
	}
	tighter := base
	tighter.Interval = int(history.IntervalBounds.Most)
	if got := learnedAt(t, s, tighter); got[0] >= standing[0] {
		t.Errorf("an interval of a year learns %v of what an interval of three weeks learns %v",
			got, standing)
	}
}

// The day the whole material stands learned is drawn where a preset is steered
// by its minutes and where it is steered by its retention, and nowhere under a
// date: a preset aiming at a day is answered by that day.
//
// It is a prediction under a stated assumption — that every card asked comes
// back — so it is read off a run of its own and not off the run beside it.
func TestTheDayTheMaterialIsLearnedIsDrawnUnderMinutesAndRetention(t *testing.T) {
	s := opened(t, studied(30))
	for back := 60; back >= 15; back -= 15 {
		record := s.run(t, noon.AddDate(0, 0, -back))
		for i := range 20 {
			answer(t, record, mark(i), 6*time.Second)
		}
	}

	p := history.Defaults()
	p.MinutesADay, p.NewADay, p.ReviewsADay = 20, 4, 60
	p.Rule, p.Interval = history.RuleInterval, 21

	for _, goal := range []history.Goal{history.GoalMinutes, history.GoalRetention} {
		one := p
		one.Goal = goal
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one)
		if err != nil {
			t.Fatal(err)
		}
		named := false
		for i, place := range got.At {
			if place.Learns == history.LearnsUnasked {
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
	dated.Goal = history.GoalDate
	dated.By = noon.AddDate(0, 0, 40).Truncate(24 * time.Hour)
	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", dated)
	if err != nil {
		t.Fatal(err)
	}
	for i, place := range got.At {
		if place.Learns != history.LearnsUnasked {
			t.Errorf("aiming at a day, %v names day %d as well", got.Grid[i], place.Learns)
		}
	}
}

// The day the file names is a place of the grid, and the mark stands on it.
//
// The picture puts the mark on the range and the person reads the numbers
// beneath it, so those numbers are the day they named and not the day beside
// it.
func TestTheMarkOfADateStandsOnTheDayTheFileNames(t *testing.T) {
	s := opened(t, studied(30))
	for _, days := range []int{1, 2, 7, 14, 21, 30, 90, 365} {
		p := history.Preset{
			Goal: history.GoalDate, By: noon.AddDate(0, 0, days).Truncate(24 * time.Hour),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Rule: history.RuleRetention, Retention: 0.9,
		}

		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.At < 0 {
			t.Fatalf("a date %d days off stands nowhere on its own range", days)
		}
		if got.Days[got.Now.At] != got.Now.Day {
			t.Errorf("a date %d days off is marked at %s and the place under the mark is %s",
				days, got.Now.Day, got.Days[got.Now.At])
		}
		if got.Grid[got.Now.At] != got.Now.Value {
			t.Errorf("a date %d days off is marked at %v days and the place under the mark "+
				"is %v days", days, got.Now.Value, got.Grid[got.Now.At])
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
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 30).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Now.At < 0 {
		t.Fatal("the preset stands nowhere on its own curve")
	}
	stands := got.At[got.Now.At]
	if stands.Through < 1 {
		t.Fatalf("the day it aims at gets through %v of the material", stands.Through)
	}
	if stands.Owed != 0 {
		t.Errorf("the day it aims at is through the material and leaves %d card faces owed",
			stands.Owed)
	}
	if stands.Retained < 0.5 {
		t.Errorf("the day it aims at gets through the material and leaves %v of it in the head",
			stands.Retained)
	}
}

// A place of a date's range is one run, and everything read off it agrees.
//
// The height of the curve and the mark on it answered out of two runs: the
// height came from the preset's own date, which schedules nothing past the day
// it names, so every place to the right of that day drew a material falling out
// of the head while the mark beside it said the day was met.
func TestAPlaceOfADateIsOneRun(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 5).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.At {
		if one.Enough != one.Met {
			t.Errorf("%s is got through by the budget %t and by the pace %t",
				got.Days[i], one.Enough, one.Met)
		}
		if i > 0 && got.At[i-1].Enough && !one.Enough {
			t.Errorf("%s is got through and %s, a day later, is not",
				got.Days[i-1], got.Days[i])
		}
	}
}

// A day at none of the load is no sitting under a date either, so the curve
// draws the day after it.
func TestACurveOfADateOnADayAtNoneOfTheLoadDrawsTheNextSitting(t *testing.T) {
	s := opened(t, studied(30))
	if noon.Weekday() != time.Monday {
		t.Fatalf("the day these curves are drawn on is a %v", noon.Weekday())
	}
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 20).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleRetention, Retention: 0.9,
		Load: map[time.Weekday]int{time.Monday: 0},
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range got.At {
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
	s := opened(t, studied(30))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(0, 0, 25).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleInterval, Interval: 21, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.Suggested == flashcards.Nowhere {
		t.Fatal("no day is suggested, and the range holds days the material is through by")
	}
	stands := got.At[got.Suggested.At]
	if stands.Short != 0 || !stands.Enough || stands.Through < 1 {
		t.Errorf("%s is suggested, and it leaves %d card faces out of reach and gets "+
			"through %v of the material", got.Suggested.Day, stands.Short, stands.Through)
	}
	// And it is the soonest such day.
	for i, one := range got.At {
		if i < got.Suggested.At && one.Short == 0 && one.Enough &&
			one.Minutes <= float64(p.MinutesADay) {
			t.Errorf("%s is suggested and %s, earlier, is through the material too",
				got.Suggested.Day, got.Days[i])
		}
	}
}

// A preset aiming at a day and naming none schedules nothing.
//
// The budget its goal names is the day, and a goal that cannot read its own
// budget is the pause a budget of zero is. Every overdue card face was handed
// over instead, with no count and no minutes to close the day.
func TestADateNamingNoDaySchedulesNothing(t *testing.T) {
	p := history.Preset{
		Goal: history.GoalDate, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
	}
	if !p.Paused(today, noon) {
		t.Error("a preset aiming at a day and naming none schedules something")
	}
	admits := p.Admits(today, noon, history.Spent{}, history.Left{New: 30})
	if !admits.Paused {
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
	s := opened(t, studied(2))
	p := history.Preset{
		Goal: history.GoalDate, By: noon.AddDate(20, 0, 0).Truncate(24 * time.Hour),
		MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		Rule: history.RuleRetention, Retention: 0.9,
	}

	got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Grid) == 0 {
		t.Fatal("a day twenty years off draws no range at all, as a day already past does")
	}
	if got.Now != flashcards.Nowhere {
		t.Errorf("a day twenty years off stands at %+v on a range reaching %v days",
			got.Now, got.Grid[len(got.Grid)-1])
	}
}

// The curve opens on the day a person is already partway through.
//
// The first place of a run is a real day and not a fresh one: what it draws is
// what a sitting opened now would hand over. An evening re-opening of the tab
// otherwise draws a day already spent as a day still to come.
func TestACurveDrawsWhatIsLeftOfTheDay(t *testing.T) {
	s := opened(t, studied(30))
	// The preset the deck points at says what the curve is asked for, so the
	// sitting and the picture are held to one day.
	write(t, s, "Sanskrit.md", "---\ntype: preset\ngoal: minutes_a_day\n"+
		"minutes_a_day: 3\nnew_a_day: 0\nreviews_a_day: 0\n---\n\n# Sanskrit\n")
	// Every card face owed today, at six seconds an answer.
	before := s.run(t, noon.AddDate(0, 0, -30))
	for i := range 30 {
		answer(t, before, mark(i), 6*time.Second)
	}
	p := history.Preset{
		Goal: history.GoalMinutes, MinutesADay: 3, NewADay: 0, ReviewsADay: 0,
	}

	fresh, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	// Ten of them answered this morning, which is a minute of the three.
	morning := s.run(t, noon)
	for i := range 10 {
		answer(t, morning, mark(i), 6*time.Second)
	}
	after, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}

	if after.At[after.Now.At].Reviews >= fresh.At[fresh.Now.At].Reviews {
		t.Errorf("a day with a minute of it spent draws %v cards and an unspent day %v",
			after.At[after.Now.At].Reviews, fresh.At[fresh.Now.At].Reviews)
	}
	sat := s.under(t, today, noon, "Sanskrit.md")
	if got := float64(len(sat.Asked)); got != after.At[after.Now.At].Reviews {
		t.Errorf("the sitting offers %v card faces and the curve draws %v",
			got, after.At[after.Now.At].Reviews)
	}
}

// The setting a person is standing at is a place of the grid, so the figures
// under the mark are the figures of that setting.
//
// The mark carried the preset's own value while the figures beside it were the
// grid's nearest place, so a day of fifteen minutes was drawn as a day of
// fourteen and a day of one minute as a day of two.
func TestTheMarkStandsOnTheSettingThePresetHolds(t *testing.T) {
	s := opened(t, studied(30))
	before := s.run(t, noon.AddDate(0, 0, -30))
	for i := range 30 {
		answer(t, before, mark(i), 6*time.Second)
	}

	for _, minutes := range []int{1, 3, 15, 20, 200} {
		p := history.Preset{
			Goal: history.GoalMinutes, MinutesADay: minutes, NewADay: 8, ReviewsADay: 45,
		}
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.At < 0 {
			t.Errorf("a day of %d minutes stands nowhere on its own curve", minutes)
			continue
		}
		if got.Grid[got.Now.At] != float64(minutes) {
			t.Errorf("a day of %d minutes is marked at the place drawing %v minutes",
				minutes, got.Grid[got.Now.At])
		}
	}

	for _, share := range []float64{0.71, 0.87, 0.9, 0.98} {
		p := history.Preset{
			Goal: history.GoalRetention, Retention: share,
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
		}
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
		if err != nil {
			t.Fatal(err)
		}
		if got.Now.At < 0 || got.Grid[got.Now.At] != share {
			t.Errorf("a target of %v is marked at place %d, which draws %v",
				share, got.Now.At, got.Grid[got.Now.At])
		}
	}
}
