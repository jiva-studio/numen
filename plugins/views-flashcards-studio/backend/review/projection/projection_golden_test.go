package projection_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review/projection"
)

// update writes the projections down again, and compares against none of them.
var update = flag.Bool("update", false, "write testdata/projection.golden again")

// The projection is the arithmetic every picture and every session rests on, so
// what it answers over a spread of materials and settings is written down whole
// — every scalar and every series, day by day — and compared against on every
// run.
//
// A change meaning to move a number moves this file with it, under -update, and
// the diff is what it moved.
func TestAProjectionAnswersWhatItIsSavedDownAs(t *testing.T) {
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

// goldenDay counts days in UTC, so the golden file is identical on every machine.
var goldenDay = review.Day{Starts: review.DayStartTime, In: time.UTC}

// goldenNow is the instant the projections open on.
var goldenNow = time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)

// projections is every projection this file holds, written out.
func projections(t *testing.T) string {
	t.Helper()
	delta := review.NewFSRS()
	var out strings.Builder
	for _, m := range goldenMaterials(delta) {
		for _, p := range goldenPresets() {
			run := projection.Simulation{
				By: delta, Day: goldenDay, Days: 60, Retains: goldenDays(60),
				Cost: review.AnswerCost{New: 19 * time.Second, Review: 7 * time.Second},
				Spent: review.Spent{
					Answered: 4, New: 1, Reviews: 3, Took: 40 * time.Second,
				},
			}
			// A retention goal runs under the scheduler configured for that retention.
			if p.preset.Goal == review.GoalRetention {
				run.By = review.NewFSRSAt(p.preset.Retention)
			}
			fmt.Fprintf(&out, "== %s / %s\n", m.name, p.name)
			ran, err := run.Run(t.Context(), goldenNow, p.preset, m.schedules, m.unreviewed)
			if err != nil {
				t.Fatal(err)
			}
			writeProjection(&out, ran)

			// The day the whole material is learned is computed on a second run of the
			// same input, with full recall, in which nothing is forgotten.
			run.Recalls = review.GetFullRecall
			nothing, err := run.Run(t.Context(), goldenNow, p.preset, m.schedules, m.unreviewed)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&out, "learns, nothing forgotten %d\n", nothing.Learns)
		}
	}
	return out.String()
}

// goldenDays lists every day of a run, which are the days the projections
// report the returning share for, so the golden file records the complete
// output of a run.
func goldenDays(days int) []int {
	out := make([]int, days)
	for i := range out {
		out[i] = i
	}
	return out
}

// writeProjection puts one projection down, every scalar and every series of
// it.
func writeProjection(out *strings.Builder, p projection.Projection) {
	fmt.Fprintf(out, "days %d cards %d seen %d owed %d clears %d learned %d learns %d short %d\n",
		p.Days, p.ReviewCards, p.Seen, p.Owed, p.Clears, p.Learned, p.Learns, p.Short)
	fmt.Fprintf(out, "answered %d reviews %.6f minutes %.6f admits %d\n",
		p.Answered, p.ReviewsADay, p.MinutesADay, p.CountAdmitted())
	fmt.Fprintf(out, "load %s\n", numbers(p.Load))
	fmt.Fprintf(out, "backlog %s\n", numbers(p.Backlog))
	fmt.Fprintf(out, "spent %s\n", formatSpent(p.Spent))
	fmt.Fprintf(out, "admitted %s\n", formatAdmitted(p.Admitted))
	fmt.Fprintf(out, "closed %s\n", formatClosed(p.Closed))
	fmt.Fprintf(out, "through %s\n", formatShares(p.Through))
	fmt.Fprintf(out, "retained %s\n", formatShares(getRetainedShares(p.Retained)))
}

// getRetainedShares is the share that came back on each day the run answers
// for, in order.
func getRetainedShares(one projection.RetentionByDay) []float64 {
	days := one.Days()
	out := make([]float64, 0, len(days))
	for _, day := range days {
		share, _ := one.GetShare(day)
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

func formatSpent(one []time.Duration) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = each.String()
	}
	return strings.Join(out, ",")
}

func formatAdmitted(one []bool) string {
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

func formatClosed(one []review.BudgetNames) string {
	out := make([]string, len(one))
	for i, each := range one {
		named := make([]string, len(each))
		for j, name := range each {
			named[j] = string(name)
		}
		out[i] = strings.Join(named, "+")
		if out[i] == "" {
			out[i] = "-"
		}
	}
	return strings.Join(out, ",")
}

func formatShares(one []float64) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = fmt.Sprintf("%.9f", each)
	}
	return strings.Join(out, ",")
}

// goldenMaterial is one vault a projection is drawn over.
type goldenMaterial struct {
	name       string
	schedules  map[review.CardID]review.Schedule
	unreviewed int
}

// goldenMaterials is the set of vaults the projections are computed over:
// an empty vault, a material with no begun cards, a large debt, a material
// answered in varied ways, and a vault where the day ends with both overdue
// cards and unbegun cards remaining, where the split of the day's budget
// between them determines the result.
func goldenMaterials(delta review.Scheduler) []goldenMaterial {
	return []goldenMaterial{
		{name: "an empty vault", schedules: nil, unreviewed: 0},
		{name: "nothing begun", schedules: nil, unreviewed: 40},
		{name: "a small debt", schedules: goldenAt(delta, 25, 3), unreviewed: 5},
		{name: "a large debt", schedules: goldenAt(delta, 90, 40), unreviewed: 60},
		{name: "more than a day holds of each", schedules: goldenAt(delta, 120, 120), unreviewed: 200},
	}
}

// goldenAt is a material in which each card was answered a varying number of
// times, some long overdue and some due in the coming weeks.
//
// The overdue part has its due day set before the answer that produced it,
// which a real vault never holds: a due day is computed forward from the time
// of an answer. It is a stress case, and the figures recorded for "a large
// debt" do not represent a real vault.
func goldenAt(delta review.Scheduler, reviewCards, overdue int) map[review.CardID]review.Schedule {
	out := make(map[review.CardID]review.Schedule, reviewCards)
	for i := range reviewCards {
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
			c = delta.Next(c, when, rating)
			when = c.Due
		}
		if i < overdue {
			// A review card whose due day has passed, by more days the later it appears
			// in the material.
			c.Due = goldenNow.AddDate(0, 0, -1-i%30)
		}
		out[review.CardID{Card: fmt.Sprintf("card%06d", i), Template: "Recognise"}] = c
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
			Counts: review.BudgetUnitCards, Backlog: 100, Load: light, IsEvenLoad: true,
		}},
		{"minutes, no even load, new material first", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 35, NewADay: 12, ReviewsADay: 60,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 10,
			Counts: review.BudgetUnitShows, Backlog: 0,
		}},
		{"minutes, learned by a chance of recall", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.87, Rule: review.RuleRetention,
			Counts: review.BudgetUnitCards, Backlog: 100, IsEvenLoad: true,
		}},
		{"minutes, a pause", review.Preset{
			Goal: review.GoalMinutes, MinutesADay: 0, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 21, IsEvenLoad: true,
		}},
		{"retention, asking much of memory", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.97, Rule: review.RuleInterval, Interval: 21,
			Counts: review.BudgetUnitCards, Backlog: 60, Load: light, IsEvenLoad: true,
		}},
		{"retention, counting showings", review.Preset{
			Goal: review.GoalRetention, MinutesADay: 20, NewADay: 6, ReviewsADay: 30,
			Retention: 0.97, Rule: review.RuleInterval, Interval: 21,
			Counts: review.BudgetUnitShows, Backlog: 60, Load: light, IsEvenLoad: true,
		}},
		{"a date, forty-five days off", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.BudgetUnitCards, IsEvenLoad: true,
		}},
		{"a date, counting showings", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.BudgetUnitShows, IsEvenLoad: true,
		}},
		{"a date, learned by a chance of recall", review.Preset{
			Goal: review.GoalDate, By: goldenNow.AddDate(0, 0, 20),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.85, Rule: review.RuleRetention,
			Counts: review.BudgetUnitCards, Load: light,
		}},
	}
}
