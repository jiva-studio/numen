package review_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// update writes the projections down again, and compares against none of them.
var update = flag.Bool("update", false, "write testdata/projection.golden again")

// The projection is the arithmetic every picture and every sitting rests on, so
// what it answers over a spread of materials and settings is written down whole
// — every scalar and every series, day by day — and compared against on every
// run.
//
// A change meaning to move a number moves this file with it, under -update, and
// the diff is what it moved.
func TestAProjectionAnswersWhatItIsWrittenDownAs(t *testing.T) {
	got := projections(t)
	path := filepath.Join("testdata", "projection.golden")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got == string(want) {
		return
	}
	lines, wanted := strings.Split(got, "\n"), strings.Split(string(want), "\n")
	for i := range max(len(lines), len(wanted)) {
		one, other := line(lines, i), line(wanted, i)
		if one != other {
			t.Fatalf("line %d of the projections reads\n%s\nwant\n%s", i+1, one, other)
		}
	}
}

// line is one line of a reading, and empty past the end of it.
func line(lines []string, i int) string {
	if i >= len(lines) {
		return ""
	}
	return lines[i]
}

// goldenDay is the day of review the projections are counted in. It stands in
// UTC so that the file reads the same on every machine.
var goldenDay = review.Day{Starts: review.DayStarts, In: time.UTC}

// goldenNow is the instant the projections open on.
var goldenNow = time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)

// projections is every projection this file holds, written out.
func projections(t *testing.T) string {
	t.Helper()
	by := review.NewFSRS()
	var out strings.Builder
	for _, m := range goldenMaterials(by) {
		for _, p := range goldenPresets() {
			run := review.Simulation{
				By: by, Day: goldenDay, Days: 60, Retains: goldenDays(60),
				Cost: review.AnswerCost{New: 19 * time.Second, Review: 7 * time.Second},
				Spent: review.Spent{
					Answered: 4, New: 1, Reviews: 3, Took: 40 * time.Second,
				},
			}
			// A goal of retention runs under the scheduler it asks for.
			if p.preset.Goal == review.GoalRetention {
				run.By = review.NewFSRSAt(p.preset.Retention)
			}
			fmt.Fprintf(&out, "== %s / %s\n", m.name, p.name)
			ran, err := run.Run(t.Context(), goldenNow, p.preset, m.at, m.unseen)
			if err != nil {
				t.Fatal(err)
			}
			written(&out, ran)

			// The day the whole material stands learned is drawn on a second run
			// of the same place, in which nothing is forgotten.
			run.Recalls = review.NothingForgotten
			nothing, err := run.Run(t.Context(), goldenNow, p.preset, m.at, m.unseen)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&out, "learns, nothing forgotten %d\n", nothing.Learns)
		}
	}
	return out.String()
}

// goldenDays is every day of a run, which is what the projections are asked to
// answer the returning share for: what this file writes down is the whole of
// what a run can say.
func goldenDays(days int) []int {
	out := make([]int, days)
	for i := range out {
		out[i] = i
	}
	return out
}

// written puts one projection down, every scalar and every series of it.
func written(out *strings.Builder, p review.Projection) {
	fmt.Fprintf(out, "days %d faces %d seen %d owed %d clears %d learned %d learns %d short %d\n",
		p.Days, p.Faces, p.Seen, p.Owed, p.Clears, p.Learned, p.Learns, p.Short)
	fmt.Fprintf(out, "answered %d reviews %.6f minutes %.6f admits %d\n",
		p.Answered, p.ReviewsADay, p.MinutesADay, p.Admits())
	fmt.Fprintf(out, "load %s\n", numbers(p.Load))
	fmt.Fprintf(out, "backlog %s\n", numbers(p.Backlog))
	fmt.Fprintf(out, "spent %s\n", spent(p.Spent))
	fmt.Fprintf(out, "admitted %s\n", admitted(p.Admitted))
	fmt.Fprintf(out, "closed %s\n", closed(p.Closed))
	fmt.Fprintf(out, "through %s\n", shares(p.Through))
	fmt.Fprintf(out, "retained %s\n", shares(kept(p.Retained)))
}

// kept is the share that came back on each day the run answers for, in order.
func kept(one review.Kept) []float64 {
	days := one.Days()
	out := make([]float64, 0, len(days))
	for _, day := range days {
		share, _ := one.On(day)
		out = append(out, share)
	}
	return out
}

func numbers(one []int) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = fmt.Sprint(each)
	}
	return strings.Join(out, ",")
}

func spent(one []time.Duration) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = each.String()
	}
	return strings.Join(out, ",")
}

func admitted(one []bool) string {
	var out strings.Builder
	for _, each := range one {
		if each {
			out.WriteByte('y')
			continue
		}
		out.WriteByte('n')
	}
	return out.String()
}

func closed(one []review.BudgetNames) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = strings.Join(each.Names(), "+")
		if out[i] == "" {
			out[i] = "-"
		}
	}
	return strings.Join(out, ",")
}

func shares(one []float64) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = fmt.Sprintf("%.9f", each)
	}
	return strings.Join(out, ",")
}

// goldenMaterial is one vault a projection is drawn over.
type goldenMaterial struct {
	name   string
	at     map[review.CardFaceID]review.Schedule
	unseen int
}

// goldenMaterials is the spread of vaults the projections are drawn over:
// nothing at all, a material nobody has begun, one standing on a large debt,
// one answered every which way, and one where the day runs out with both the
// debt and the unbegun material still holding cards, so that the share the day
// is split in decides what it asks for.
func goldenMaterials(by review.Scheduler) []goldenMaterial {
	return []goldenMaterial{
		{name: "an empty vault", at: nil, unseen: 0},
		{name: "nothing begun", at: nil, unseen: 40},
		{name: "a small debt", at: goldenAt(by, 25, 3), unseen: 5},
		{name: "a large debt", at: goldenAt(by, 90, 40), unseen: 60},
		{name: "more than a day holds of each", at: goldenAt(by, 120, 120), unseen: 200},
	}
}

// goldenAt is a material answered a varying number of times each, some of it
// long overdue and some of it falling due in the weeks ahead.
//
// The overdue part has its due day put behind the answer that produced it,
// which is a state a vault does not hold: a due day is worked out forward from
// the instant of an answer. It stands here as a stress case, and the figures
// written down under "a large debt" are not a picture of anybody's vault.
func goldenAt(by review.Scheduler, faces, overdue int) map[review.CardFaceID]review.Schedule {
	out := make(map[review.CardFaceID]review.Schedule, faces)
	for i := range faces {
		c := review.Schedule{}
		when := goldenNow.AddDate(0, 0, -90+i%17)
		for step := range 1 + i%6 {
			rating := review.Good
			if step > 0 && (i+step)%5 == 0 {
				rating = review.Again
			}
			if step > 0 && (i+step)%7 == 0 {
				rating = review.Easy
			}
			c = by.Next(c, when, rating)
			when = c.Due
		}
		if i < overdue {
			// A card face whose day has come and gone, by more days the further
			// down the material it stands.
			c.Due = goldenNow.AddDate(0, 0, -1-i%30)
		}
		out[review.CardFaceID{Card: fmt.Sprintf("card%06d", i), Face: "Recognise"}] = c
	}
	return out
}

// goldenPreset is one set of settings a projection is drawn under.
type goldenPreset struct {
	name   string
	preset review.Preset
}

// goldenPresets is the spread of settings: each goal, each rule for what counts
// as learned, a week with a light day and a day at nothing, a day spent new
// material first, a preset that schedules nothing, and each counting of what a
// day's budget is spent on set against the same settings at the other.
func goldenPresets() []goldenPreset {
	light := map[time.Weekday]int{time.Saturday: 50, time.Sunday: 0}
	return []goldenPreset{
		{"minutes, the defaults", review.Defaults()},
		{"minutes, a light week", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21,
			Counts: review.CountsCards, Backlog: 100, Load: light, EvenLoad: true,
		}},
		{"minutes, no even load, new material first", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 35, NewADay: 12, ReviewsADay: 60,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 10,
			Counts: review.CountsShows, Backlog: 0,
		}},
		{"minutes, learned by a chance of recall", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.87, Rule: review.RuleRetention,
			Counts: review.CountsCards, Backlog: 100, EvenLoad: true,
		}},
		{"minutes, a pause", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 0, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21, EvenLoad: true,
		}},
		{"retention, asking much of memory", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.97, Rule: review.RuleInterval, Interval: 21,
			Counts: review.CountsCards, Backlog: 60, Load: light, EvenLoad: true,
		}},
		{"retention, counting showings", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.97, Rule: review.RuleInterval, Interval: 21,
			Counts: review.CountsShows, Backlog: 60, Load: light, EvenLoad: true,
		}},
		{"a date, forty-five days off", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.CountsCards, EvenLoad: true,
		}},
		{"a date, counting showings", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.CountsShows, EvenLoad: true,
		}},
		{"a date, learned by a chance of recall", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 20),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.85, Rule: review.RuleRetention,
			Counts: review.CountsCards, Load: light,
		}},
	}
}
