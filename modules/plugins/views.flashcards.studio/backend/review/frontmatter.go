package review

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
)

// ReadPreset is what a preset note's frontmatter says, and what could not be
// read in it.
//
// A key that cannot be read is a problem against the note and keeps its
// default, and the file is never repaired.
// readGoal parses the goal field from frontmatter.
func readGoal(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["goal"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "goal is not text")
	case IsKnownGoal(Goal(name)):
		p.Goal = Goal(name)
	default:
		*problems = append(*problems, "goal "+name+" is not minutes_a_day, retention or by_date")
	}
}

// readLearnedRule parses the learned field from frontmatter.
func readLearnedRule(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["learned"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "learned is not text")
	case IsKnownRule(LearnedRule(name)):
		p.Rule = LearnedRule(name)
	default:
		*problems = append(*problems, "learned "+name+" is not interval or retention")
	}
}

// readBudgetUnit parses the counts field from frontmatter.
func readBudgetUnit(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["counts"]
	if !present || raw == nil {
		return
	}
	name, isText := raw.(string)
	switch {
	case !isText:
		*problems = append(*problems, "counts is not text")
	case IsKnownBudgetUnit(BudgetUnit(name)):
		p.Counts = BudgetUnit(name)
	default:
		*problems = append(*problems, "counts "+name+" is not cards or shows")
	}
}

// readByDate parses the by_date field from frontmatter.
func readByDate(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["by_date"]
	if !present || raw == nil {
		if p.Goal == GoalDate && p.By.IsZero() {
			*problems = append(*problems, "a preset aiming at a day says which day, under by_date")
		}
		return
	}
	switch value := raw.(type) {
	case time.Time:
		p.By = value
	case string:
		day, err := time.Parse(DayLayout, strings.TrimSpace(value))
		if err != nil {
			*problems = append(*problems, "by_date "+value+" is not a day, written as "+DayLayout)
			break
		}
		p.By = day
	default:
		*problems = append(*problems, "by_date is not a day")
	}
	if p.Goal == GoalDate && p.By.IsZero() {
		*problems = append(*problems, "a preset aiming at a day says which day, under by_date")
	}
}

// readSchedule parses goal, learned rule, counts unit, and target date from frontmatter.
func readSchedule(front map[string]any, p *Preset, problems *[]string) {
	readGoal(front, p, problems)
	readLearnedRule(front, p, problems)
	readBudgetUnit(front, p, problems)
	readByDate(front, p, problems)
}

// readLimits parses review limits and intervals from frontmatter.
func readLimits(front map[string]any, p *Preset, problems *[]string) {
	p.MinutesADay = readWholeNumber(front, "minutes_a_day", MinutesADayBounds, p.MinutesADay, problems)
	p.NewADay = readWholeNumber(front, "new_a_day", NewADayBounds, p.NewADay, problems)
	p.ReviewsADay = readWholeNumber(front, "reviews_a_day", ReviewsADayBounds, p.ReviewsADay, problems)
	p.Backlog = readWholeNumber(front, "backlog", BacklogBounds, p.Backlog, problems)
	p.Interval = readWholeNumber(front, "interval", IntervalBounds, p.Interval, problems)
}

// readRetention parses the retention target from frontmatter.
func readRetention(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["retention"]
	if !present || raw == nil {
		return
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, "retention is not a number")
	case !RetentionBounds.Contains(value):
		*problems = append(*problems, fmt.Sprintf(
			"retention %g is outside %g to %g", value, RetentionBounds.Least, RetentionBounds.Most))
	default:
		p.Retention = value
	}
}

// readEvenLoad parses the even_load boolean flag from frontmatter.
func readEvenLoad(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["even_load"]
	if !present || raw == nil {
		return
	}
	value, isBool := raw.(bool)
	if !isBool {
		*problems = append(*problems, "even_load is neither true nor false")
		return
	}
	p.IsEvenLoad = value
}

// readDayShare parses a single weekday load percentage from frontmatter.
func readDayShare(name string, raw any, p *Preset, daysLen int, problems *[]string) {
	day, recognized := Weekday(name)
	if !recognized {
		*problems = append(*problems, "the load of "+name+" is not a day of the week")
		return
	}
	share, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, "the load of "+name+" is not a number")
	case share != math.Trunc(share):
		*problems = append(*problems, "the load of "+name+" is counted in whole per cent")
	case !LoadBounds.Contains(share):
		*problems = append(*problems, fmt.Sprintf("the load of %s, %g, is outside %g to %g",
			name, share, LoadBounds.Least, LoadBounds.Most))
	default:
		if p.Load == nil {
			p.Load = make(map[time.Weekday]int, daysLen)
		}
		p.Load[day] = int(share)
	}
}

// readLoad parses weekly load distribution from frontmatter.
func readLoad(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["load"]
	if !present || raw == nil {
		return
	}
	days, isMapping := raw.(map[string]any)
	if !isMapping {
		*problems = append(*problems, "load is a day of the week against a share of a day's load")
		return
	}
	// A map hands its keys over in whatever order it holds them, and a
	// note read twice says the same both times.
	for _, name := range slices.Sorted(maps.Keys(days)) {
		readDayShare(name, days[name], p, len(days), problems)
	}
}

// readWeights parses retention, even_load, and weekday load distribution from frontmatter.
func readWeights(front map[string]any, p *Preset, problems *[]string) {
	readRetention(front, p, problems)
	readEvenLoad(front, p, problems)
	readLoad(front, p, problems)
}

func ReadPreset(front map[string]any) (Preset, []string) {
	p := Defaults()
	var problems []string

	readSchedule(front, &p, &problems)
	readLimits(front, &p, &problems)
	readWeights(front, &p, &problems)

	return p, problems
}

// weekdays are the days of the week as a preset writes them.
var weekdays = map[string]time.Weekday{
	"mon": time.Monday,
	"tue": time.Tuesday,
	"wed": time.Wednesday,
	"thu": time.Thursday,
	"fri": time.Friday,
	"sat": time.Saturday,
	"sun": time.Sunday,
}

// Weekday is the day one of those names, and whether it names one.
func Weekday(name string) (time.Weekday, bool) {
	day, recognized := weekdays[strings.ToLower(strings.TrimSpace(name))]
	return day, recognized
}

// DayName is how a preset writes a day of the week.
func DayName(day time.Weekday) string {
	return strings.ToLower(day.String()[:3])
}

// readWholeNumber is one setting written in whole numbers, and what the note
// said about it.
func readWholeNumber(
	front map[string]any, key string, bounds Bounds, fallback int, problems *[]string,
) int {
	raw, present := front[key]
	if !present || raw == nil {
		return fallback
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, key+" is not a number")
	case value != math.Trunc(value):
		*problems = append(*problems, key+" is counted in whole numbers")
	case !bounds.Contains(value):
		*problems = append(*problems, fmt.Sprintf(
			"%s %g is outside %g to %g", key, value, bounds.Least, bounds.Most))
	default:
		return int(value)
	}
	return fallback
}

// number is what YAML hands over for a number, whichever of the two it chose.
func number(raw any) (float64, bool) {
	switch value := raw.(type) {
	case int:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}
