package review

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"
)

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

func readByDate(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["by_date"]
	if !present || raw == nil {
		if p.Goal == GoalDate && p.By.IsZero() {
			*problems = append(*problems, "a preset targeting a day requires the day, under by_date")
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
		*problems = append(*problems, "a preset targeting a day requires the day, under by_date")
	}
}

func readSchedule(front map[string]any, p *Preset, problems *[]string) {
	readGoal(front, p, problems)
	readLearnedRule(front, p, problems)
	readBudgetUnit(front, p, problems)
	readByDate(front, p, problems)
}

func readLimits(front map[string]any, p *Preset, problems *[]string) {
	p.MinutesADay = readWholeNumber(front, "minutes_a_day", GetMinutesADayBounds(), p.MinutesADay, problems)
	p.NewADay = readWholeNumber(front, "new_a_day", GetNewADayBounds(), p.NewADay, problems)
	p.ReviewsADay = readWholeNumber(front, "reviews_a_day", GetReviewsADayBounds(), p.ReviewsADay, problems)
	p.Backlog = readWholeNumber(front, "backlog", GetBacklogBounds(), p.Backlog, problems)
	p.Interval = readWholeNumber(front, "interval", GetIntervalBounds(), p.Interval, problems)
}

func readRetention(front map[string]any, p *Preset, problems *[]string) {
	raw, present := front["retention"]
	if !present || raw == nil {
		return
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, "retention is not a number")
	case !GetRetentionBounds().Contains(value):
		*problems = append(*problems, fmt.Sprintf(
			"retention %g is outside %g to %g", value, GetRetentionBounds().Min, GetRetentionBounds().Max))
	default:
		p.Retention = value
	}
}

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
	case !GetLoadBounds().Contains(share):
		*problems = append(*problems, fmt.Sprintf("the load of %s, %g, is outside %g to %g",
			name, share, GetLoadBounds().Min, GetLoadBounds().Max))
	default:
		if p.Load == nil {
			p.Load = make(map[time.Weekday]int, daysLen)
		}
		p.Load[day] = int(share)
	}
}

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
	// A map iterates in arbitrary order, so the keys are sorted and a note read
	// twice yields the same result.
	for _, name := range slices.Sorted(maps.Keys(days)) {
		readDayShare(name, days[name], p, len(days), problems)
	}
}

func readWeights(front map[string]any, p *Preset, problems *[]string) {
	readRetention(front, p, problems)
	readEvenLoad(front, p, problems)
	readLoad(front, p, problems)
}

// ReadPreset reads a preset from frontmatter and reports the problems found.
func ReadPreset(front map[string]any) (Preset, []string) {
	p := Defaults()
	var problems []string

	readSchedule(front, &p, &problems)
	readLimits(front, &p, &problems)
	readWeights(front, &p, &problems)

	return p, problems
}

// Weekday is the day of the week a preset writes as the name, and whether the name is one.
func Weekday(name string) (time.Weekday, bool) {
	for day := time.Sunday; day <= time.Saturday; day++ {
		if DayName(day) == strings.ToLower(strings.TrimSpace(name)) {
			return day, true
		}
	}
	// gomutants:disable-next-line RETURN_ZERO reason="Constant time.Sunday is already the zero Weekday value."
	return time.Sunday, false
}

// DayName is how a preset writes a day of the week.
func DayName(day time.Weekday) string {
	return strings.ToLower(day.String()[:3])
}

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
			"%s %g is outside %g to %g", key, value, bounds.Min, bounds.Max))
	default:
		return int(value)
	}
	return fallback
}

func number(raw any) (float64, bool) {
	switch value := raw.(type) {
	case int:
		return float64(value), true
	case float64:
		return value, true
	}
	return 0, false
}
