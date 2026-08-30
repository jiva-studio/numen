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
// with it. What is suggested is the target leaving the most in the head.
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

	best := got.Suggested.At
	for i, one := range got.At {
		if one.Retained > got.At[best].Retained+1e-9 {
			t.Errorf("%v is suggested and retains %v, while %v retains %v",
				got.Grid[best], got.At[best].Retained, got.Grid[i], one.Retained)
		}
	}
}

// A goal of a date is a finite range: every day from today to the day named,
// each carrying what a day of review has to run to be through by then. A later
// day never costs more than an earlier one.
func TestTheCurveOfADateRunsToTheDayNamed(t *testing.T) {
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
	if last := got.Days[len(got.Days)-1]; last != p.By.Format(history.Named) {
		t.Errorf("the curve runs to %q, and the day named is %q", last, p.By.Format(history.Named))
	}
	if got.Now.At != len(got.Grid)-1 {
		t.Errorf("the preset stands at place %d of %d", got.Now.At, len(got.Grid))
	}

	for i := 1; i < len(got.At); i++ {
		was, now := got.At[i-1], got.At[i]
		if was.Met && now.Minutes > was.Minutes {
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

// A day nothing gets the material through by is said so, and the budget that
// falls short is told how much of it it would get through.
func TestADayThatCannotBeMet(t *testing.T) {
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
		if one.Met || one.Enough {
			t.Errorf("%s is met at %+v, and one new card a day is three cards of thirty",
				got.Days[i], one)
		}
		if one.Through <= 0 || one.Through >= 1 {
			t.Errorf("%s gets through %v of the material", got.Days[i], one.Through)
		}
	}
	if got.Suggested != flashcards.Nowhere {
		t.Errorf("suggested %+v, and no day on the curve can be met", got.Suggested)
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

// Light days and an even load are projected, so what stands where the preset
// stands is worked out with them applied.
func TestTheCurveIsWorkedOutWithLightDaysAndAnEvenLoad(t *testing.T) {
	s := opened(t, studied(30))
	p := history.Preset{Goal: history.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}

	flat, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", p)
	if err != nil {
		t.Fatal(err)
	}
	light, even := p, p
	light.LightDays = []time.Weekday{time.Wednesday, time.Sunday}
	even.EvenLoad = true

	for name, one := range map[string]history.Preset{"light days": light, "an even load": even} {
		got, err := s.curves(noon).Execute(t.Context(), s.vault, "Sanskrit.md", one)
		if err != nil {
			t.Fatal(err)
		}
		inRange(t, got)
		if got.Now.At != flat.Now.At {
			t.Fatalf("with %s the preset stands at place %d, and without at %d",
				name, got.Now.At, flat.Now.At)
		}
		if got.At[got.Now.At] == flat.At[flat.Now.At] {
			t.Errorf("with %s the preset comes to %+v, the same as a preset keeping neither",
				name, got.At[got.Now.At])
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
