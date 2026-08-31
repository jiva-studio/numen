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

	// Counts is what a day's budget is spent on: the cards a day holds, or the
	// times they are put to a person.
	Counts Counts

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

// Counts is what a day's budget is spent on.
//
// Under CountsCards a card face is charged the first time it is answered in a
// review day and comes round again in it for nothing. Under CountsShows every
// showing is charged.
type Counts string

const (
	CountsCards Counts = "cards"
	CountsShows Counts = "shows"
)

// KnownCounts reports whether a value is one of the two.
func KnownCounts(c Counts) bool {
	switch c {
	case CountsCards, CountsShows:
		return true
	}
	return false
}

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
		Counts:      CountsCards,
		EvenLoad:    true,
	}
}

// Closes is which of a preset's budgets closes its day.
//
// The goal names the budget that closes the day, and every other budget takes
// no part. A budget taking no part stands in the file where the person left it
// and is in force again the moment its goal is chosen.
type Closes struct {
	New     bool
	Reviews bool
	Minutes bool
}

// Allowance is what one day of a preset admits: how many cards of each kind it
// has room for, how long the day still runs, and which of the three closes it.
//
// It is the one answer to what a day admits. A sitting spends against it and a
// projection runs on it, so the picture a person drags a control over is the
// arithmetic the sitting will run.
type Allowance struct {
	// Keeps is what the preset keeps for the whole of this day, the day of the
	// week having had its say.
	Keeps Budget
	// New, Reviews and Minutes are what is left of it.
	New     int
	Reviews int
	Minutes time.Duration
	// Closes is which of the three closes the day.
	Closes Closes
	// Paused is a preset that schedules nothing at all.
	Paused bool
}

// Admits is what this preset's day admits.
//
// Now is any instant of the review day, spent is what that day has already gone
// through under the preset, and left is how much of the material the preset has
// still to begin.
func (p Preset) Admits(d Day, now time.Time, spent Spent, left int) Allowance {
	opened := d.Ends(now).AddDate(0, 0, -1)
	out := Allowance{
		Keeps:  p.on(opened.Weekday()),
		Closes: p.closing(),
		Paused: p.Paused(d, now),
	}
	if p.Goal == GoalDate {
		out.Keeps.New = p.paces(d, now, left)
	}
	out.New = out.Keeps.New - spent.New
	out.Reviews = out.Keeps.Reviews - spent.Reviews
	out.Minutes = time.Duration(out.Keeps.Minutes*float64(time.Minute)) - spent.Took
	return out
}

// closing is which budget closes this preset's day.
//
// A goal of a date closes the day on a count of new cards, which is the share
// of the material a day has to begin to be through it by then.
func (p Preset) closing() Closes {
	switch p.Goal {
	case GoalRetention:
		return Closes{New: true, Reviews: true}
	case GoalDate:
		return Closes{New: true}
	default:
		return Closes{Minutes: true}
	}
}

// Paused reports whether the preset schedules nothing: the budget its goal
// names is zero, or a day that has passed.
func (p Preset) Paused(d Day, now time.Time) bool {
	if p.Past(d, now) {
		return true
	}
	switch p.Goal {
	case GoalRetention:
		return p.NewADay == 0 && p.ReviewsADay == 0
	case GoalDate:
		return false
	default:
		return p.MinutesADay == 0
	}
}

// paces is how much of the material a day holds when a date sets the pace: what
// is left to begin, over the days left to begin it in. A day past the one it
// aims at holds none of it.
func (p Preset) paces(d Day, now time.Time, left int) int {
	days := p.days(d, now)
	if days <= 0 {
		return 0
	}
	return (left + days - 1) / days
}

// days is how many days of review there are from the day holding now through to
// the day this preset aims at, counting both. A preset aiming at no day, or at
// one behind us, has none.
func (p Preset) days(d Day, now time.Time) int {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(Named, d.Names(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	out := int(to.Sub(from).Hours()/24) + 1
	if out < 0 {
		return 0
	}
	return out
}

// Past reports whether the review day the goal names is behind us. The day it
// names is a whole day of review, and a preset aiming at no day is never past.
func (p Preset) Past(d Day, now time.Time) bool {
	if p.Goal != GoalDate || p.By.IsZero() {
		return false
	}
	return !now.Before(d.Ending(p.By))
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

	if raw, present := front["counts"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "counts is not text")
		case KnownCounts(Counts(name)):
			p.Counts = Counts(name)
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
