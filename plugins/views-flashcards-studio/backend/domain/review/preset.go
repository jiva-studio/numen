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
type Preset struct {
	Goal Goal
	By   time.Time

	MinutesADay int
	NewADay     int
	ReviewsADay int
	Retention   float64

	Rule     LearnedRule
	Interval int

	Counts BudgetUnit

	Backlog int

	Load       map[time.Weekday]int
	IsEvenLoad bool
}

// FullLoad is a whole day's load, which is what a day the preset does not name
// carries.
const FullLoad = 100

// GetShare is how much of a day's load this day of the week carries, as a share
// of one.
func (p Preset) GetShare(day time.Weekday) float64 {
	per, named := p.Load[day]
	if !named {
		return 1
	}
	return float64(per) / FullLoad
}

// ShouldBalanceLoad reports whether this preset moves a review card off the day the scheduler
// chose.
//
// A preset aiming at a day does not. The pace is what spreads a date's material
// over its days, and the days it has are the days it needs.
func (p Preset) ShouldBalanceLoad() bool { return p.IsEvenLoad && p.Goal != GoalDate }

// GetPlacing is how this preset puts a card on a day, as a short name: whether
// it evens the days out, and the share each day of the week carries. A schedule
// worked out under one placing is not read back under another.
func (p Preset) GetPlacing() string {
	var out strings.Builder
	fmt.Fprintf(&out, "even=%t", p.ShouldBalanceLoad())
	for day := time.Sunday; day <= time.Saturday; day++ {
		fmt.Fprintf(&out, " %s=%d", DayName(day), int(math.Round(p.GetShare(day)*FullLoad)))
	}
	return out.String()
}

// Goal is which value the one control steers.
type Goal string

const (
	// GoalMinutes steers the day by minutes of review.
	GoalMinutes Goal = "minutes_a_day"
	// GoalRetention steers the day by the target retention.
	GoalRetention Goal = "retention"
	// GoalDate steers the day by the date the material is to be learned by.
	GoalDate Goal = "by_date"
)

// IsKnownGoal reports whether a goal is one of the three.
func IsKnownGoal(g Goal) bool {
	switch g {
	case GoalMinutes, GoalRetention, GoalDate:
		return true
	}
	return false
}

// LearnedRule is the criterion by which a preset counts a card as learned. The
// value is stored under the key the rule names, and the other rule keeps its value and takes no part.
type LearnedRule string

const (
	// RuleInterval learns a review card once it is sent away for the preset's
	// interval or longer.
	RuleInterval LearnedRule = "interval"
	// RuleRetention learns a review card once the chance of recalling it today is
	// at or above the preset's retention.
	RuleRetention LearnedRule = "retention"
)

// IsKnownRule reports whether a rule is one of the two.
func IsKnownRule(r LearnedRule) bool {
	switch r {
	case RuleInterval, RuleRetention:
		return true
	}
	return false
}

// IsLearned reports whether a review card standing at this schedule is one the
// person has learned at this instant, under the rule this preset names.
//
// It is the one place the rule is read.
func (p Preset) IsLearned(s Schedule, target time.Time) bool {
	if s.IsNew() {
		return false
	}
	rule, interval, retention := p.GetLearnedRule()
	if rule == RuleRetention {
		return Recall(target.Sub(s.Last), s.Stability) >= retention
	}
	return s.Due.Sub(s.Last) >= time.Duration(interval)*24*time.Hour
}

// GetLearnedRule is the rule a review card is counted learned by here, and the
// two values a rule reads.
//
// A preset naming no rule counts by the default rule, and a value the rule
// to the threshold in Defaults.
func (p Preset) GetLearnedRule() (LearnedRule, int, float64) {
	defaults := Defaults()
	rule, interval, retention := p.Rule, p.Interval, p.Retention
	if !IsKnownRule(rule) {
		rule = defaults.Rule
	}
	if !GetIntervalBounds().Contains(float64(interval)) {
		interval = defaults.Interval
	}
	if !GetRetentionBounds().Contains(retention) {
		retention = defaults.Retention
	}
	return rule, interval, retention
}

// BudgetUnit is what a day's budget is counted in.
//
// Under BudgetUnitCards a review card is charged the first time it is answered in
// a review day and comes round again in it for nothing. Under BudgetUnitShows
// every showing is charged.
type BudgetUnit string

const (
	// BudgetUnitCards counts each review card once a day.
	BudgetUnitCards BudgetUnit = "cards"
	// BudgetUnitShows counts every showing of a review card.
	BudgetUnitShows BudgetUnit = "shows"
)

// IsCharged reports whether a showing of a review card spends a slot of a day's
// count, where isShown reports whether the day has already presented that review card.
//
// It is the one place the counting is read.
func (u BudgetUnit) IsCharged(isShown bool) bool { return u == BudgetUnitShows || !isShown }

// IsKnownBudgetUnit reports whether a value is one of the two.
func IsKnownBudgetUnit(u BudgetUnit) bool {
	switch u {
	case BudgetUnitCards, BudgetUnitShows:
		return true
	}
	return false
}

// Bounds is how far a setting goes, at each end.
type Bounds struct{ Min, Max float64 }

// Contains reports whether a value is within the bounds.
func (b Bounds) Contains(value float64) bool { return value >= b.Min && value <= b.Max }

// What each setting of a preset may be. A day holds no more minutes than it
// has, a retention target outside these is a scheduler asking for what memory
// does not do, and an interval a card is learned at is a day at the least and a
// year at the most.

// GetMinutesADayBounds returns the bounds of the minutes a day.
func GetMinutesADayBounds() Bounds { return Bounds{Min: 0, Max: 24 * 60} }

// GetNewADayBounds returns the bounds of the new cards a day.
func GetNewADayBounds() Bounds { return Bounds{Min: 0, Max: 9999} }

// GetReviewsADayBounds returns the bounds of the reviews a day.
func GetReviewsADayBounds() Bounds { return Bounds{Min: 0, Max: 9999} }

// GetRetentionBounds returns the bounds of the retention target.
func GetRetentionBounds() Bounds { return Bounds{Min: 0.7, Max: 0.99} }

// GetIntervalBounds returns the bounds of the interval a card is learned at, in days.
func GetIntervalBounds() Bounds { return Bounds{Min: 1, Max: 365} }

// GetBacklogBounds returns the bounds of the backlog.
func GetBacklogBounds() Bounds { return Bounds{Min: 0, Max: 100} }

// GetLoadBounds returns the bounds of the load.
func GetLoadBounds() Bounds { return Bounds{Min: 0, Max: FullLoad} }

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
		Counts:      BudgetUnitCards,
		Backlog:     AllBacklog,
		IsEvenLoad:  true,
		By:          time.Time{},
		Load:        nil,
	}
}
