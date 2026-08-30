package flashcards

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Preset is how the decks pointing at one note are scheduled.
//
// It is read from the frontmatter of a note of `type: preset`. A key the file
// does not carry stands at the default, and no cards a day is a pause.
type Preset struct {
	// Goal is the value the one control steers. The value itself is in the
	// field that goal names.
	Goal Goal
	// By is the day the material is to be in the head, and is read when the
	// goal is GoalDate.
	By time.Time

	// MinutesADay is how long a day of review runs, spent against the time each
	// answer took. Zero is a preset keeping no budget in time.
	MinutesADay int
	// NewADay and ReviewsADay are how many cards of each kind a day holds.
	NewADay     int
	ReviewsADay int
	// Retention is the share of cards recalled when they come round again.
	Retention float64

	// LightDays are the days of the week the load is cut on, and the cards
	// moved to their neighbours.
	LightDays []time.Weekday
	// EvenLoad is whether days are made to resemble each other.
	EvenLoad bool
}

// Goal is which value the one control steers.
type Goal string

const (
	GoalMinutes   Goal = "minutes_a_day"
	GoalRetention Goal = "retention"
	GoalDate      Goal = "by_date"
)

// KnownGoal reports whether a goal is one of the three.
func KnownGoal(g Goal) bool {
	switch g {
	case GoalMinutes, GoalRetention, GoalDate:
		return true
	}
	return false
}

// DayFormat is how the day a goal names is written.
const DayFormat = "2006-01-02"

// Bounds is how far a setting goes, at each end.
type Bounds struct{ Least, Most float64 }

// Holds reports whether a value is within the bounds.
func (b Bounds) Holds(value float64) bool { return value >= b.Least && value <= b.Most }

// What each setting of a preset may be. A day holds no more minutes than it
// has, and a retention target outside these is a scheduler asking for what
// memory does not do.
var (
	MinutesADayBounds = Bounds{Least: 0, Most: 24 * 60}
	NewADayBounds     = Bounds{Least: 0, Most: 9999}
	ReviewsADayBounds = Bounds{Least: 0, Most: 9999}
	RetentionBounds   = Bounds{Least: 0.7, Most: 0.99}
)

// Defaults is a preset naming nothing, and how a deck pointing at no preset is
// scheduled.
func Defaults() Preset {
	return Preset{
		Goal:        GoalMinutes,
		MinutesADay: 20,
		NewADay:     10,
		ReviewsADay: 200,
		Retention:   0.9,
		EvenLoad:    true,
	}
}

// Paused reports whether the preset schedules nothing: no cards a day, or a day
// that has passed.
func (p Preset) Paused(today time.Time) bool {
	return (p.NewADay == 0 && p.ReviewsADay == 0) || p.Spent(today)
}

// Spent reports whether the day the goal names is behind us. A preset aiming at
// no day is never spent.
func (p Preset) Spent(today time.Time) bool {
	if p.Goal != GoalDate || p.By.IsZero() {
		return false
	}
	return today.After(p.By)
}

// ReadPreset is what a preset note's frontmatter says, and what could not be
// read in it.
//
// A key that cannot be read is a problem against the note and keeps its
// default: the file is never repaired, because repairing means guessing at
// what the person wrote.
func ReadPreset(front map[string]any) (Preset, []string) {
	p := Defaults()
	var problems []string

	if raw, present := front["goal"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "goal is not text")
		case KnownGoal(Goal(name)):
			p.Goal = Goal(name)
		default:
			problems = append(problems, "goal "+name+" is not minutes_a_day, retention or by_date")
		}
	}

	if raw, present := front["by_date"]; present && raw != nil {
		switch value := raw.(type) {
		case time.Time:
			p.By = value
		case string:
			day, err := time.Parse(DayFormat, strings.TrimSpace(value))
			if err != nil {
				problems = append(problems, "by_date "+value+" is not a day, written as "+DayFormat)
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

	p.MinutesADay = counted(front, "minutes_a_day", MinutesADayBounds, p.MinutesADay, &problems)
	p.NewADay = counted(front, "new_a_day", NewADayBounds, p.NewADay, &problems)
	p.ReviewsADay = counted(front, "reviews_a_day", ReviewsADayBounds, p.ReviewsADay, &problems)

	if raw, present := front["retention"]; present && raw != nil {
		value, ok := number(raw)
		switch {
		case !ok:
			problems = append(problems, "retention is not a number")
		case !RetentionBounds.Holds(value):
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
			p.EvenLoad = value
		}
	}

	if raw, present := front["light_days"]; present && raw != nil {
		days, isList := raw.([]any)
		if !isList {
			problems = append(problems, "light_days is a list of days")
		} else {
			for _, one := range days {
				name, isText := one.(string)
				if !isText {
					problems = append(problems, "a light day is not text")
					continue
				}
				day, known := Weekday(name)
				if !known {
					problems = append(problems, "light day "+name+" is not a day of the week")
					continue
				}
				p.LightDays = append(p.LightDays, day)
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

// counted is one setting written in whole numbers, and what the note said
// about it.
func counted(
	front map[string]any, key string, bounds Bounds, standing int, problems *[]string,
) int {
	raw, present := front[key]
	if !present || raw == nil {
		return standing
	}
	value, ok := number(raw)
	switch {
	case !ok:
		*problems = append(*problems, key+" is not a number")
	case value != math.Trunc(value):
		*problems = append(*problems, key+" is counted in whole numbers")
	case !bounds.Holds(value):
		*problems = append(*problems, fmt.Sprintf(
			"%s %g is outside %g to %g", key, value, bounds.Least, bounds.Most))
	default:
		return int(value)
	}
	return standing
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
