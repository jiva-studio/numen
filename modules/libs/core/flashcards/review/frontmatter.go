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
func ReadPreset(front map[string]any) (Preset, []string) {
	p := Defaults()
	var problems []string

	if raw, present := front["goal"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "goal is not text")
		case IsKnownGoal(Goal(name)):
			p.Goal = Goal(name)
		default:
			problems = append(problems, "goal "+name+" is not minutes_a_day, retention or by_date")
		}
	}

	if raw, present := front["learned"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "learned is not text")
		case IsKnownRule(LearnedRule(name)):
			p.Rule = LearnedRule(name)
		default:
			problems = append(problems, "learned "+name+" is not interval or retention")
		}
	}

	if raw, present := front["counts"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "counts is not text")
		case IsKnownBudgetUnit(BudgetUnit(name)):
			p.Counts = BudgetUnit(name)
		default:
			problems = append(problems, "counts "+name+" is not cards or shows")
		}
	}

	if raw, present := front["by_date"]; present && raw != nil {
		switch value := raw.(type) {
		case time.Time:
			p.By = value
		case string:
			day, err := time.Parse(Named, strings.TrimSpace(value))
			if err != nil {
				problems = append(problems, "by_date "+value+" is not a day, written as "+Named)
				break
			}
			p.By = day
		default:
			problems = append(problems, "by_date is not a day")
		}
	}
	if p.Goal == GoalDate && p.By.IsZero() {
		problems = append(problems, "a preset aiming at a day says which day, under by_date")
	}

	p.MinutesADay = readWholeNumber(front, "minutes_a_day", MinutesADayBounds, p.MinutesADay, &problems)
	p.NewADay = readWholeNumber(front, "new_a_day", NewADayBounds, p.NewADay, &problems)
	p.ReviewsADay = readWholeNumber(front, "reviews_a_day", ReviewsADayBounds, p.ReviewsADay, &problems)
	p.Backlog = readWholeNumber(front, "backlog", BacklogBounds, p.Backlog, &problems)
	p.Interval = readWholeNumber(front, "interval", IntervalBounds, p.Interval, &problems)

	if raw, present := front["retention"]; present && raw != nil {
		value, ok := number(raw)
		switch {
		case !ok:
			problems = append(problems, "retention is not a number")
		case !RetentionBounds.Contains(value):
			problems = append(problems, fmt.Sprintf(
				"retention %g is outside %g to %g", value, RetentionBounds.Least, RetentionBounds.Most))
		default:
			p.Retention = value
		}
	}

	if raw, present := front["even_load"]; present && raw != nil {
		value, isBool := raw.(bool)
		if !isBool {
			problems = append(problems, "even_load is neither true nor false")
		} else {
			p.IsEvenLoad = value
		}
	}

	if raw, present := front["load"]; present && raw != nil {
		days, isMapping := raw.(map[string]any)
		if !isMapping {
			problems = append(problems, "load is a day of the week against a share of a day's load")
		} else {
			// A map hands its keys over in whatever order it holds them, and a
			// note read twice says the same both times.
			for _, name := range slices.Sorted(maps.Keys(days)) {
				day, known := Weekday(name)
				if !known {
					problems = append(problems, "the load of "+name+" is not a day of the week")
					continue
				}
				share, ok := number(days[name])
				switch {
				case !ok:
					problems = append(problems, "the load of "+name+" is not a number")
				case share != math.Trunc(share):
					problems = append(problems, "the load of "+name+" is counted in whole per cent")
				case !LoadBounds.Contains(share):
					problems = append(problems, fmt.Sprintf("the load of %s, %g, is outside %g to %g",
						name, share, LoadBounds.Least, LoadBounds.Most))
				default:
					if p.Load == nil {
						p.Load = make(map[time.Weekday]int, len(days))
					}
					p.Load[day] = int(share)
				}
			}
		}
	}

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
	day, known := weekdays[strings.ToLower(strings.TrimSpace(name))]
	return day, known
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
