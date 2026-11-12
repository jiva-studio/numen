package projection_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/projection"
)

// updateFlag names the flag that writes the projections down again, and compares against none of
// them.
const updateFlag = "update"

func TestMain(m *testing.M) {
	flag.Bool(updateFlag, false, "write testdata/projection.golden again")
	flag.Parse()
	os.Exit(m.Run())
}

func isUpdating() bool {
	return flag.Lookup(updateFlag).Value.String() == "true"
}

// The projection is the arithmetic every picture and every session rests on, so what it answers
// over a spread of materials and settings is written down whole, every scalar and every series
// day by day, and compared on every run. A change meant to move a number moves this file with
// it, under -update, and the diff is what it moved.
func TestAProjectionAnswersWhatItIsSavedDownAs(t *testing.T) {
	got := projections(t)
	path := filepath.Join("testdata", "projection.golden")
	if isUpdating() {
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

func line(lines []string, i int) string {
	if i >= len(lines) {
		return ""
	}
	return lines[i]
}

func getGoldenDay() review.Day {
	return review.Day{Starts: review.DayStartTime, In: time.UTC}
}

func getGoldenNow() time.Time {
	return time.Date(2026, 3, 2, 9, 41, 0, 0, time.UTC)
}

func projections(t *testing.T) string {
	t.Helper()
	delta := review.NewFSRS()
	var out strings.Builder
	for _, m := range goldenMaterials(delta) {
		for _, p := range goldenPresets() {
			run := projection.Simulation{
				By: delta, Day: getGoldenDay(), Days: 60, Retains: goldenDays(60),
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
			ran, err := run.Run(t.Context(), getGoldenNow(), p.preset, m.schedules, m.unreviewed)
			if err != nil {
				t.Fatal(err)
			}
			writeProjection(&out, ran)

			// The day the whole material is learned is computed on a second run of the
			// same input, with full recall, in which nothing is forgotten.
			run.Recalls = review.GetFullRecall
			nothing, err := run.Run(t.Context(), getGoldenNow(), p.preset, m.schedules, m.unreviewed)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(&out, "learns, nothing forgotten %d\n", nothing.Learns)
		}
	}
	return out.String()
}

func goldenDays(days int) []int {
	out := make([]int, days)
	for i := range out {
		out[i] = i
	}
	return out
}

func writeProjection(out *strings.Builder, p projection.Projection) {
	fmt.Fprintf(out, "days %d cards %d seen %d due %d clears %d learned %d learns %d short %d\n",
		p.Days, p.ReviewCards, p.Seen, p.Due, p.Clears, p.Learned, p.Learns, p.Short)
	fmt.Fprintf(out, "answered %d reviews %.6f minutes %.6f admits %d\n",
		p.Answered, p.ReviewsADay, p.MinutesADay, p.CountAdmitted())
	fmt.Fprintf(out, "load %s\n", numbers(p.Load))
	fmt.Fprintf(out, "backlog %s\n", numbers(p.Backlog))
	fmt.Fprintf(out, "spent %s\n", formatSpentDurations(p.Spent))
	fmt.Fprintf(out, "admitted %s\n", formatAdmittedFlags(p.Admitted))
	fmt.Fprintf(out, "closed %s\n", formatClosedBudgets(p.Closed))
	fmt.Fprintf(out, "through %s\n", formatShares(p.Through))
	fmt.Fprintf(out, "retained %s\n", formatShares(getRetainedShares(p.Retained)))
}

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

func formatSpentDurations(one []time.Duration) string {
	out := make([]string, len(one))
	for i, each := range one {
		out[i] = each.String()
	}
	return strings.Join(out, ",")
}

func formatAdmittedFlags(one []bool) string {
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

func formatClosedBudgets(one []review.BudgetNames) string {
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

func goldenMaterials(delta review.Scheduler) []goldenMaterial {
	return []goldenMaterial{
		{name: "an empty vault", schedules: nil, unreviewed: 0},
		{name: "nothing begun", schedules: nil, unreviewed: 40},
		{name: "a small debt", schedules: goldenAt(delta, 25, 3), unreviewed: 5},
		{name: "a large debt", schedules: goldenAt(delta, 90, 40), unreviewed: 60},
		{name: "more than a day holds of each", schedules: goldenAt(delta, 120, 120), unreviewed: 200},
	}
}

func goldenAt(delta review.Scheduler, reviewCards, overdue int) map[review.CardID]review.Schedule {
	out := make(map[review.CardID]review.Schedule, reviewCards)
	for i := range reviewCards {
		c := review.Schedule{}
		when := getGoldenNow().AddDate(0, 0, -90+i%17)
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
			c.Due = getGoldenNow().AddDate(0, 0, -1-i%30)
		}
		out[review.CardID{Card: fmt.Sprintf("card%06d", i), Template: "Recognize"}] = c
	}
	return out
}

// goldenPreset is one set of settings a projection is drawn under.
type goldenPreset struct {
	name   string
	preset review.Preset
}

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
			Goal: review.GoalDate, By: getGoldenNow().AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.BudgetUnitCards, IsEvenLoad: true,
		}},
		{"a date, counting showings", review.Preset{
			Goal: review.GoalDate, By: getGoldenNow().AddDate(0, 0, 45),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.9, Rule: review.RuleInterval, Interval: 7,
			Counts: review.BudgetUnitShows, IsEvenLoad: true,
		}},
		{"a date, learned by a chance of recall", review.Preset{
			Goal: review.GoalDate, By: getGoldenNow().AddDate(0, 0, 20),
			MinutesADay: 20, NewADay: 8, ReviewsADay: 45,
			Retention: 0.85, Rule: review.RuleRetention,
			Counts: review.BudgetUnitCards, Load: light,
		}},
	}
}
