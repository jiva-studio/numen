package projection

import "time"

// Settings is how a preset schedules, as plain values. Load is the percentage of a full day each day of the week
// carries, and a day it does not name carries a full day. IsEveryShowingCharged spends a slot of the day's count on
// every showing of a card, and not only on the first one of the day.
type Settings struct {
	Goal                  Goal
	By                    time.Time
	MinutesADay           int
	NewADay               int
	ReviewsADay           int
	Retention             float64
	Rule                  Rule
	Interval              int
	IsEveryShowingCharged bool
	Backlog               int
	Load                  map[time.Weekday]int
	IsEvenLoad            bool
}

// Goal is the value the one control of a preset steers.
type Goal string

const (
	GoalMinutes   Goal = "minutes_a_day"
	GoalRetention Goal = "retention"
	GoalDate      Goal = "by_date"
)

// Rule is how a card is counted as learned.
type Rule string

const (
	// RuleInterval counts a card learned once it is sent away for the preset's interval or longer.
	RuleInterval Rule = "interval"
	// RuleRetention counts a card learned while its chance of recall is at or above the preset's retention.
	RuleRetention Rule = "retention"
)

// FullLoad is the percentage of a whole day's load.
const FullLoad = 100

// GetShare returns how much of a day's load this day of the week carries, as a share of one.
func (s Settings) GetShare(day time.Weekday) float64 {
	per, named := s.Load[day]
	if !named {
		return 1
	}
	return float64(per) / FullLoad
}

func (s Settings) isCharged(isShown bool) bool { return s.IsEveryShowingCharged || !isShown }

// MinutesInDay is the most minutes a review day holds.
const MinutesInDay = 24 * 60

// NeverClears is a pace that leaves something overdue on every day projected.
const NeverClears = -1

// NeverLearns is a day further off than the run looked: a run whose horizon ends with a card still to learn.
const NeverLearns = -1

// LearnsUnasked is a run with no day on which the whole material is learned.
const LearnsUnasked = -2
