package flashcards

import (
	"fmt"
	"maps"
	"math"
	"slices"
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

	// Rule is what counts as a card face the person has learned, written under
	// `learned`. The value it reads stands in the field the rule names:
	// Interval under RuleInterval, Retention under RuleRetention.
	Rule Rule
	// Interval is how long a card face is sent away for before it is learned, in
	// days, and is read under RuleInterval.
	Interval int

	// Counts is what a day's budget is spent on: the cards a day holds, or the
	// times they are put to a person.
	Counts Counts

	// Backlog is how much of a day goes to what is overdue before anything
	// unbegun is offered, as a share in hundredths. At a hundred the debt is
	// paid first and new cards are begun on what is left; at nothing the new
	// material comes first; between them the day is split, and a side that runs
	// short leaves the rest to the other.
	//
	// It says what a day is spent on rather than what closes it, and it is read
	// where one pot is spent between the two.
	Backlog int

	// Load is how much of a day's load each day of the week carries, in per
	// cent. A day the preset does not name carries the whole of it, and a day
	// at nothing schedules nothing.
	Load map[time.Weekday]int
	// EvenLoad is whether days are made to resemble each other.
	EvenLoad bool
}

// FullLoad is a whole day's load, which is what a day the preset does not name
// carries.
const FullLoad = 100

// Share is how much of a day's load this day of the week carries, as a share of
// one.
func (p Preset) Share(day time.Weekday) float64 {
	per, named := p.Load[day]
	if !named {
		return 1
	}
	return float64(per) / FullLoad
}

// Placing is how this preset puts a card on a day, as a short name: whether it
// evens the days out, and the share each day of the week carries. A schedule
// worked out under one placing is not read back under another.
func (p Preset) Placing() string {
	var out strings.Builder
	fmt.Fprintf(&out, "even=%t", p.EvenLoad)
	for day := time.Sunday; day <= time.Saturday; day++ {
		fmt.Fprintf(&out, " %s=%d", DayName(day), int(math.Round(p.Share(day)*FullLoad)))
	}
	return out.String()
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

// Rule is what a preset counts as learned. The value it reads stands under the
// key it names, and the other rule keeps its value and takes no part.
type Rule string

const (
	// RuleInterval learns a card face once it is sent away for the preset's
	// interval or longer.
	RuleInterval Rule = "interval"
	// RuleRetention learns a card face once the chance of recalling it today is
	// at or above the preset's retention.
	RuleRetention Rule = "retention"
)

// KnownRule reports whether a rule is one of the two.
func KnownRule(r Rule) bool {
	switch r {
	case RuleInterval, RuleRetention:
		return true
	}
	return false
}

// Learned reports whether a card face standing at this schedule is one the
// person has learned at this instant, under the rule this preset names.
//
// A card face nobody has answered is learned by neither rule.
//
// It is the one place the rule is read, so a projection and everything drawn
// beside it answer the same question.
func (p Preset) Learned(s Schedule, at time.Time) bool {
	if !s.Seen() {
		return false
	}
	rule, interval, retention := p.counting()
	if rule == RuleRetention {
		return Recall(at.Sub(s.Last), s.Stability) >= retention
	}
	return s.Due.Sub(s.Last) >= time.Duration(interval)*24*time.Hour
}

// counting is the rule a card face is counted learned by here, and the two
// values a rule reads.
//
// A preset naming no rule counts by the default rule, and a value the rule
// cannot hold stands at the default. A preset that says nothing holds its cards
// to the threshold in Defaults, which is the threshold the docs write down.
func (p Preset) counting() (Rule, int, float64) {
	standing := Defaults()
	rule, interval, retention := p.Rule, p.Interval, p.Retention
	if !KnownRule(rule) {
		rule = standing.Rule
	}
	if !IntervalBounds.Holds(float64(interval)) {
		interval = standing.Interval
	}
	if !RetentionBounds.Holds(retention) {
		retention = standing.Retention
	}
	return rule, interval, retention
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

// AllBacklog is a day spent on the debt before anything unbegun is offered,
// which is what a preset naming no share does.
const AllBacklog = 100

// Bounds is how far a setting goes, at each end.
type Bounds struct{ Least, Most float64 }

// Holds reports whether a value is within the bounds.
func (b Bounds) Holds(value float64) bool { return value >= b.Least && value <= b.Most }

// What each setting of a preset may be. A day holds no more minutes than it
// has, a retention target outside these is a scheduler asking for what memory
// does not do, and an interval a card is learned at is a day at the least and a
// year at the most.
var (
	MinutesADayBounds = Bounds{Least: 0, Most: 24 * 60}
	NewADayBounds     = Bounds{Least: 0, Most: 9999}
	ReviewsADayBounds = Bounds{Least: 0, Most: 9999}
	RetentionBounds   = Bounds{Least: 0.7, Most: 0.99}
	IntervalBounds    = Bounds{Least: 1, Most: 365}
	BacklogBounds     = Bounds{Least: 0, Most: 100}
	LoadBounds        = Bounds{Least: 0, Most: FullLoad}
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
		Rule:        RuleInterval,
		Interval:    21,
		Counts:      CountsCards,
		Backlog:     AllBacklog,
		EvenLoad:    true,
	}
}

// Closes is what each of a preset's settings governs under the goal in force,
// each written as the preset writes the key, and empty where it takes no part.
//
// A setting taking no part stands in the file where the person left it and is
// in force again the moment its goal is chosen.
type Closes struct {
	// New, Reviews and Minutes are the budgets. The goal names the one that
	// closes the day, and the others take no part.
	New     Closed
	Reviews Closed
	Minutes Closed
	// Backlog is the share of the day that goes to the debt. It closes nothing:
	// it says what the day is spent on, and it is empty under a goal whose day
	// is not one pot spent between the two. A goal of a date carries the whole
	// material by its own reckoning, and a goal of retention holds each side to
	// a count of its own.
	Backlog Closed
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
	// Backlog is how much of the day goes to the debt before anything unbegun
	// is offered, as a share in hundredths. A goal whose day is not one pot
	// spent between the two stands at the whole of it, and the debt is paid
	// first.
	Backlog int
	// Paused is a preset that schedules nothing at all.
	Paused bool
}

// Left is what a preset has still to get through: the card faces nobody has
// begun, and how many days of review one begun now needs before the preset
// counts it learned, which is Ripens and which a date paces the day against.
type Left struct {
	New    int
	Ripens int
}

// Admits is what this preset's day admits.
//
// Now is any instant of the review day, spent is what that day has already gone
// through under the preset, and left is the material it has still to begin.
func (p Preset) Admits(d Day, now time.Time, spent Spent, left Left) Allowance {
	opened := d.Ends(now).AddDate(0, 0, -1)
	out := Allowance{
		Keeps:  p.on(opened.Weekday()),
		Closes: p.closing(),
		// A day carrying none of the load schedules nothing, as a budget of
		// zero does.
		Paused: p.Paused(d, now) || p.Share(opened.Weekday()) == 0,
	}
	if p.Goal == GoalDate {
		// The pace is a whole day's share of the material, and this day carries
		// as much of it as its day of the week carries of the load.
		out.Keeps.New = int(math.Round(p.Share(opened.Weekday()) * float64(p.paces(d, now, left))))
	}
	// A split that takes no part leaves the day spending on the debt first,
	// which is what a preset naming no share does.
	out.Backlog = AllBacklog
	if out.Closes.Backlog != ClosedNothing {
		out.Backlog = p.Backlog
	}
	out.New = out.Keeps.New - spent.New
	out.Reviews = out.Keeps.Reviews - spent.Reviews
	out.Minutes = time.Duration(out.Keeps.Minutes*float64(time.Minute)) - spent.Took
	return out
}

// Paying reports whether the next card of this day comes from the debt before
// it rather than from the material it has not begun.
//
// Debt and begun are how many of each the day has taken so far, and owed and
// fresh whether either side has a card left to give. The day is spent between
// the two in the share the preset names; a side with nothing left leaves the
// rest of the day to the other, and an odd card goes to the debt.
//
// It is the one rule for how a day is spent, so a sitting and a projection of
// that same day put the same cards in the same order.
func (a Allowance) Paying(debt, begun int, owed, fresh bool) bool {
	if !owed {
		return false
	}
	if !fresh {
		return true
	}
	return debt*AllBacklog < a.Backlog*(debt+begun+1)
}

// closing is which budget closes this preset's day.
//
// A goal of a date closes the day on a count of new cards, which is the share
// of the material a day has to begin to be through it by then. The date is what
// paced that count, so the date is what closed the day.
//
// The share of the day that goes to the debt is read where one pot is spent
// between the two. A goal of retention keeps a count for each side, so each is
// held to its own and the share decides nothing.
func (p Preset) closing() Closes {
	switch p.Goal {
	case GoalRetention:
		return Closes{New: ClosedNew, Reviews: ClosedReviews}
	case GoalDate:
		return Closes{New: ClosedDate}
	default:
		return Closes{Minutes: ClosedMinutes, Backlog: ClosedBacklog}
	}
}

// Paused reports whether the preset schedules nothing: the budget its goal
// names is zero or absent, or a day that has passed.
func (p Preset) Paused(d Day, now time.Time) bool {
	if p.Past(d, now) {
		return true
	}
	switch p.Goal {
	case GoalRetention:
		return p.NewADay == 0 && p.ReviewsADay == 0
	case GoalDate:
		// A preset aiming at a day that names none has no budget at all, and a
		// budget the goal names and cannot read is a pause.
		return p.By.IsZero()
	default:
		return p.MinutesADay == 0
	}
}

// paces is how much of the material a day holds when a date sets the pace: what
// is left to begin, over the days on which beginning a card still leaves it time
// to be learned by the day the preset aims at. A day past the one it aims at
// holds none of it.
//
// Where no day leaves that much time, the pace is everything left. It is the
// pace that gets there every card face that can, and how many cannot is
// Projection.Short.
func (p Preset) paces(d Day, now time.Time, left Left) int {
	days := p.days(d, now)
	if days <= 0 {
		return 0
	}
	in := days - float64(left.Ripens)
	if left.Ripens == NeverRipens || in <= 0 {
		return left.New
	}
	return int(math.Ceil(float64(left.New) / in))
}

// days is how many days of review there are from the day holding now through to
// the day this preset aims at, counting both. A preset aiming at no day, or at
// one behind us, has none.
//
// Each day counts for the share of the load its day of the week carries, so a
// day at nothing is no day of review at all and a day at half is half of one.
func (p Preset) days(d Day, now time.Time) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(Named, d.Names(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	return p.admits(from, int(to.Sub(from).Hours()/24)+1)
}

// admits is how many whole days of review the preset holds over the calendar
// days from this one.
func (p Preset) admits(from time.Time, days int) float64 {
	if days <= 0 {
		return 0
	}
	week := 0.0
	for day := time.Sunday; day <= time.Saturday; day++ {
		week += p.Share(day)
	}
	whole := days / 7
	out := week * float64(whole)
	for i := range days % 7 {
		out += p.Share(from.AddDate(0, 0, whole*7+i).Weekday())
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

	if raw, present := front["learned"]; present && raw != nil {
		name, isText := raw.(string)
		switch {
		case !isText:
			problems = append(problems, "learned is not text")
		case KnownRule(Rule(name)):
			p.Rule = Rule(name)
		default:
			problems = append(problems, "learned "+name+" is not interval or retention")
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
	p.Backlog = counted(front, "backlog", BacklogBounds, p.Backlog, &problems)
	p.Interval = counted(front, "interval", IntervalBounds, p.Interval, &problems)

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
				case !LoadBounds.Holds(share):
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
