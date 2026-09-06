package review

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
	// answer took. It closes the day under a goal of minutes, where zero is a
	// pause, and no other goal reads it.
	MinutesADay int
	// NewADay and ReviewsADay are how many cards of each kind a day holds.
	NewADay     int
	ReviewsADay int
	// Retention is the share of cards recalled when they come round again.
	Retention float64

	// Rule is what counts as a card face the person has learned, written under
	// `learned`. The value it reads stands in the field the rule names:
	// Interval under RuleInterval, Retention under RuleRetention.
	Rule LearnedRule
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
	// It says what a day is spent on, closes nothing, and is read where one pot
	// is spent between the two.
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

// Evens reports whether this preset moves a card face off the day the scheduler
// chose.
//
// A preset aiming at a day does not. The pace is what spreads a date's material
// over its days, and the days it has are the days it needs.
func (p Preset) Evens() bool { return p.EvenLoad && p.Goal != GoalDate }

// Placing is how this preset puts a card on a day, as a short name: whether it
// evens the days out, and the share each day of the week carries. A schedule
// worked out under one placing is not read back under another.
func (p Preset) Placing() string {
	var out strings.Builder
	fmt.Fprintf(&out, "even=%t", p.Evens())
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

// LearnedRule is what a preset counts as learned. The value it reads stands
// under the key it names, and the other rule keeps its value and takes no part.
type LearnedRule string

const (
	// RuleInterval learns a card face once it is sent away for the preset's
	// interval or longer.
	RuleInterval LearnedRule = "interval"
	// RuleRetention learns a card face once the chance of recalling it today is
	// at or above the preset's retention.
	RuleRetention LearnedRule = "retention"
)

// KnownRule reports whether a rule is one of the two.
func KnownRule(r LearnedRule) bool {
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
// It is the one place the rule is read.
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
// to the threshold in Defaults.
func (p Preset) counting() (LearnedRule, int, float64) {
	defaults := Defaults()
	rule, interval, retention := p.Rule, p.Interval, p.Retention
	if !KnownRule(rule) {
		rule = defaults.Rule
	}
	if !IntervalBounds.Holds(float64(interval)) {
		interval = defaults.Interval
	}
	if !RetentionBounds.Holds(retention) {
		retention = defaults.Retention
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

// Charges reports whether a showing of a card face spends a slot of a day's
// count, where shown is whether the day has asked that face already.
//
// It is the one place the counting is read.
func (c Counts) Charges(shown bool) bool { return c == CountsShows || !shown }

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

// StopReason is why a preset schedules nothing, and empty where it schedules
// something. The list is closed, and a caller maps a value to a sentence.
type StopReason string

const (
	// StoppedNothing is a preset that schedules: its decks are handed a day of
	// review.
	StoppedNothing StopReason = ""
	// StoppedNoMinutes is a goal of minutes with the minutes at zero.
	StoppedNoMinutes StopReason = "no_minutes"
	// StoppedNoCards is a goal of retention with both card counts at zero.
	StoppedNoCards StopReason = "no_cards"
	// StoppedNoDay is a goal of a date naming no day. The budget the goal names
	// is the day, and a goal that cannot read its own budget schedules nothing.
	StoppedNoDay StopReason = "no_day"
	// StoppedPastDay is a goal of a date whose day is behind us.
	StoppedPastDay StopReason = "past_day"
	// StoppedNoLoad is a day of the week carrying none of the load. It is a
	// fact about one day: the preset schedules on the days that carry some.
	StoppedNoLoad StopReason = "no_load"
	// StoppedNoWeek is a week carrying none of the load. Every day of it stands
	// at nothing, so there is no day for the cards to be picked up on.
	StoppedNoWeek StopReason = "no_week"
)

// Stops is why this preset schedules nothing, and StoppedNothing where it
// schedules something.
//
// It is the one place the rule is read. A goal of a date is answered against the
// day holding now, and stops once the day it names is behind that one. A week
// every day of which carries none of the load is read whatever the goal, since
// no budget is spent on a day that schedules nothing.
func (p Preset) Stops(d Day, now time.Time) StopReason {
	switch p.Goal {
	case GoalRetention:
		if p.NewADay == 0 && p.ReviewsADay == 0 {
			return StoppedNoCards
		}
	case GoalDate:
		if p.By.IsZero() {
			return StoppedNoDay
		}
		if p.Past(d, now) {
			return StoppedPastDay
		}
	default:
		if p.MinutesADay == 0 {
			return StoppedNoMinutes
		}
	}
	if p.Week() == 0 {
		return StoppedNoWeek
	}
	return StoppedNothing
}

// Week is how many whole days of review a week of this preset holds, counting
// each day of it for the share of the load it carries.
func (p Preset) Week() float64 {
	out := 0.0
	for day := time.Sunday; day <= time.Saturday; day++ {
		out += p.Share(day)
	}
	return out
}

// StopsOn is why this preset schedules nothing on the day holding now: whatever
// stops the preset at all, and a day of the week carrying none of the load. A
// week with no day carrying any stops the preset itself, so this day is one of
// the quiet days of a week that has loud ones.
func (p Preset) StopsOn(d Day, now time.Time) StopReason {
	if why := p.Stops(d, now); why != StoppedNothing {
		return why
	}
	if p.Share(d.Opened(now).Weekday()) == 0 {
		return StoppedNoLoad
	}
	return StoppedNothing
}

// Paused reports whether the preset schedules nothing.
func (p Preset) Paused(d Day, now time.Time) bool { return p.Stops(d, now) != StoppedNothing }

// paces is how much of the material a day holds when a date sets the pace: what
// is left to begin, over the days on which beginning a card still leaves it time
// to be learned by the day the preset aims at. A day past the one it aims at
// holds none of it.
//
// Where no day leaves that much time, the pace is everything left. It is the
// pace that gets there every card face that can, and how many cannot is
// Projection.Short.
func (p Preset) paces(d Day, now time.Time, unbegunCards, daysToLearn int) int {
	days := p.days(d, now)
	if days <= 0 {
		return 0
	}
	if daysToLearn == NeverRipens {
		return unbegunCards
	}
	in := p.beginning(d, now, daysToLearn)
	if in <= 0 {
		return unbegunCards
	}
	return int(math.Ceil(float64(unbegunCards) / in))
}

// beginning is how much room a date leaves for beginning cards: the days of
// review from the day holding now up to the last one on which a card begun
// still has its ripening before the day the preset aims at.
//
// Each day counts for the share of the load its day of the week carries, and
// the ripening is counted in days of review, so the days it takes are dropped
// at the shares they carry.
func (p Preset) beginning(d Day, now time.Time, ripens int) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(Named, d.Names(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	span := int(to.Sub(from).Hours()/24) + 1
	for span > 0 && ripens > 0 {
		span--
		if p.Share(from.AddDate(0, 0, span).Weekday()) > 0 {
			ripens--
		}
	}
	return p.admits(from, span)
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
	whole := days / 7
	out := p.Week() * float64(whole)
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
