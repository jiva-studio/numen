package review

import (
	"math"
	"time"
)

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
	// StoppedNoWeek is a week carrying none of the load. Every day of it has a share
	// of zero, so there is no day on which cards are due.
	StoppedNoWeek StopReason = "no_week"
)

// GetOverallStopReason is why this preset schedules nothing on any day, and StoppedNothing where
// it schedules something. It is the one place the rule is read. A goal of a date is answered
// against the day holding now, and stops once the day it names is behind that one. A week every
// day of which carries none of the load is read whatever the goal, since no budget is spent on a
// day that schedules nothing.
func (p Preset) GetOverallStopReason(d Day, now time.Time) StopReason {
	switch p.Goal {
	case GoalRetention:
		if p.NewADay == 0 && p.ReviewsADay == 0 {
			return StoppedNoCards
		}
	case GoalDate:
		if p.By.IsZero() {
			return StoppedNoDay
		}
		if p.IsPast(d, now) {
			return StoppedPastDay
		}
	case GoalMinutes:
		if p.MinutesADay == 0 {
			return StoppedNoMinutes
		}
	default:
		if p.MinutesADay == 0 {
			return StoppedNoMinutes
		}
	}
	if p.GetWeeklyShare() == 0 {
		return StoppedNoWeek
	}
	return StoppedNothing
}

// GetWeeklyShare is how many whole days of review a week of this preset holds, counting
// each day of it for the share of the load it carries.
func (p Preset) GetWeeklyShare() float64 {
	out := 0.0
	for day := time.Sunday; day <= time.Saturday; day++ {
		out += p.GetShare(day)
	}
	return out
}

// GetStopReason is why this preset schedules nothing on the day holding now:
// whatever stops the preset at all, and a day of the week carrying none of the
// load. A week with no day carrying any stops the preset itself, so this day is
// one of the quiet days of a week that has loud ones.
func (p Preset) GetStopReason(d Day, now time.Time) StopReason {
	if why := p.GetOverallStopReason(d, now); why != StoppedNothing {
		return why
	}
	if p.GetShare(d.GetDate(now).Weekday()) == 0 {
		return StoppedNoLoad
	}
	return StoppedNothing
}

// IsPaused reports whether the preset schedules nothing.
func (p Preset) IsPaused(d Day, now time.Time) bool {
	return p.GetOverallStopReason(d, now) != StoppedNothing
}

// NeverClears is a pace that leaves something overdue on every day projected.
const NeverClears = -1

// NeverLearns is a day further off than what was looked at: a projection whose
// horizon ends with one review card still to learn.
const NeverLearns = -1

// LearnsUnasked is a projection with no day on which the whole material is
// learned.
const LearnsUnasked = -2

func (p Preset) getPace(d Day, now time.Time, unbegunCards, daysToLearn int) int {
	reviewDays := p.getReviewDays(d, now)
	if reviewDays <= 0 {
		return 0
	}
	if daysToLearn == NeverLearns {
		return unbegunCards
	}
	in := p.getCapacityToBegin(d, now, daysToLearn)
	if in <= 0 {
		return unbegunCards
	}
	return int(math.Ceil(float64(unbegunCards) / in))
}

func (p Preset) getCapacityToBegin(d Day, now time.Time, daysToLearn int) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(DayLayout, d.GetName(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	span := int(to.Sub(from).Hours()/24) + 1
	for span > 0 && daysToLearn > 0 {
		span--
		if p.GetShare(from.AddDate(0, 0, span).Weekday()) > 0 {
			daysToLearn--
		}
	}
	return p.getDaysOfReview(from, span)
}

func (p Preset) getReviewDays(d Day, now time.Time) float64 {
	if p.Goal != GoalDate || p.By.IsZero() {
		return 0
	}
	from, err := time.Parse(DayLayout, d.GetName(now))
	if err != nil {
		return 0
	}
	y, m, day := p.By.Date()
	to := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	return p.getDaysOfReview(from, int(to.Sub(from).Hours()/24)+1)
}

func (p Preset) getDaysOfReview(from time.Time, days int) float64 {
	if days <= 0 {
		return 0
	}
	whole := days / 7
	out := p.GetWeeklyShare() * float64(whole)
	for i := range days % 7 {
		out += p.GetShare(from.AddDate(0, 0, whole*7+i).Weekday())
	}
	return out
}

// IsPast reports whether the review day the goal names is behind us. The day it
// names is a whole day of review, and a preset aiming at no day is never past.
func (p Preset) IsPast(d Day, now time.Time) bool {
	if p.Goal != GoalDate || p.By.IsZero() {
		return false
	}
	return !now.Before(d.GetEndOfDate(p.By))
}
