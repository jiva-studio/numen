package review

import (
	"math"
	"slices"
	"time"
)

// AllBacklog is a day spent on the debt before anything unbegun is offered,
// which is what a preset naming no share does.
const AllBacklog = 100

// Budget is what one day of a preset holds: how many cards of each kind, and
// how long the day runs.
type Budget struct {
	New     int
	Reviews int
	Minutes float64
}

func (p Preset) getBudget(day time.Weekday) Budget {
	share := p.GetShare(day)
	return Budget{
		New:     int(math.Round(share * float64(p.NewADay))),
		Reviews: int(math.Round(share * float64(p.ReviewsADay))),
		Minutes: share * float64(p.MinutesADay),
	}
}

// BudgetName is one budget that may stop a day of review, named by the key the
// preset writes it under. A budget the goal does not name never closes the day,
// and a day that requested everything available is closed by nothing: the
// material ran out.
type BudgetName string

const (
	// ClosedNothing is the absence of a budget that closes the day.
	ClosedNothing BudgetName = ""
	// ClosedMinutes is a day paced by minutes of review.
	ClosedMinutes BudgetName = "minutes_a_day"
	// ClosedNew is a day closed by its count of new cards.
	ClosedNew BudgetName = "new_a_day"
	// ClosedReviews is a day closed by its count of reviews.
	ClosedReviews BudgetName = "reviews_a_day"
	// ClosedDate is a day paced by the day the preset aims at.
	ClosedDate BudgetName = "by_date"
	// ClosedBacklog is the share of a day that goes to the debt. It closes nothing;
	// it is a setting that a goal either reads or leaves idle.
	ClosedBacklog BudgetName = "backlog"
	// ClosedPaused is a preset scheduling nothing at all.
	ClosedPaused BudgetName = "paused"
)

// BudgetNames is every budget that closed one day of review.
//
// A goal of retention holds a day to both card counts, and a day that exhausted
// both new cards and reviews names both: raising one of them changes nothing when
// the other also closed the day.
type BudgetNames []BudgetName

func getClosers() []BudgetName {
	return []BudgetName{ClosedPaused, ClosedMinutes, ClosedNew, ClosedReviews, ClosedDate}
}

// Has reports whether this budget is one of those that closed the day.
func (c BudgetNames) Has(one BudgetName) bool { return slices.Contains(c, one) }

// Add is this closing and one budget more, named once and in the order closers
func (c BudgetNames) Add(one BudgetName) BudgetNames {
	closers := getClosers()
	if !slices.Contains(closers, one) || c.Has(one) {
		return c
	}
	out := make(BudgetNames, 0, len(c)+1)
	for _, each := range closers {
		if each == one || c.Has(each) {
			out = append(out, each)
		}
	}
	return out
}

// GetName is the key one budget is written under, and empty where the day was
// closed by none or by more than one.
func (c BudgetNames) GetName() string {
	if len(c) != 1 {
		return ""
	}
	return string(c[0])
}

// Limits is which key caps each budget under the goal in force, each written as
// the preset writes the key, and empty where it takes no part.
//
// in force again the moment its goal is chosen.
type Limits struct {
	New     BudgetName
	Reviews BudgetName
	Minutes BudgetName
	Backlog BudgetName
}

// Allowance is what one day of a preset admits: the remaining number of cards of
// each kind, the remaining time, and which of the three budgets closes it.
//
// It is the one answer to what a day admits: a session spends against it, and a
// projection runs on it.
type Allowance struct {
	Keeps   Budget
	New     int
	Reviews int
	Minutes time.Duration
	Limits  Limits
	Backlog int
	Stops   StopReason
}

// IsPaused reports whether this day schedules nothing.
func (a Allowance) IsPaused() bool { return a.Stops != StoppedNothing }

// GetAllowance is what this preset's day admits.
// Now is any instant of the review day, spent is what that day has already gone through,
// unbegunCards is the material it has still to begin, and daysToLearn is how many days of review
// a review card begun now needs before the preset counts it learned, which a date paces against.
func (p Preset) GetAllowance(
	d Day, now time.Time, used Spent, unbegunCards, daysToLearn int,
) Allowance {
	date := d.GetDate(now)
	out := Allowance{
		Keeps: p.getBudget(date.Weekday()),
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Field New is unconditionally overwritten from Keeps and used before return."
		New: 0,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Field Reviews is unconditionally overwritten from Keeps and used before return."
		Reviews: 0,
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Field Minutes is unconditionally overwritten from Keeps and used before return."
		Minutes: 0,
		Limits:  p.getLimits(),
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Field Backlog is unconditionally overwritten before return."
		Backlog: 0,
		Stops:   p.GetStopReason(d, now),
	}
	if p.Goal == GoalDate {
		// The pace is a whole day's share of the material, and this day carries
		// as much of it as its day of the week carries of the load.
		out.Keeps.New = int(math.Round(
			p.GetShare(date.Weekday()) * float64(p.getPace(d, now, unbegunCards, daysToLearn))))
	}
	// A split that takes no part leaves the day spending on the debt first,
	// which is what a preset naming no share does.
	out.Backlog = AllBacklog
	if out.Limits.Backlog != ClosedNothing {
		out.Backlog = p.Backlog
	}
	out.New = out.Keeps.New - used.New
	out.Reviews = out.Keeps.Reviews - used.Reviews
	out.Minutes = time.Duration(out.Keeps.Minutes*float64(time.Minute)) - used.Took
	return out
}

// IsNextCardDebt reports whether the next card of this day comes from the debt before it. False
// is a card from the material the preset has not begun. Debt and begun are the counts of each
// kind taken so far; isDue and isNew report whether each side has a card left. The day is split
// in the share the preset names, a side with nothing left leaves the rest to the other, and an
// odd card goes to the debt.
func (a Allowance) IsNextCardDebt(debt, begun int, isDue, isNew bool) bool {
	if !isDue {
		return false
	}
	if !isNew {
		return true
	}
	return debt*AllBacklog < a.Backlog*(debt+begun+1)
}

func (p Preset) getLimits() Limits {
	switch p.Goal {
	case GoalRetention:
		return Limits{New: ClosedNew, Reviews: ClosedReviews, Minutes: ClosedNothing, Backlog: ClosedNothing}
	case GoalDate:
		return Limits{New: ClosedDate, Reviews: ClosedNothing, Minutes: ClosedNothing, Backlog: ClosedNothing}
	case GoalMinutes:
		return Limits{New: ClosedNothing, Reviews: ClosedNothing, Minutes: ClosedMinutes, Backlog: ClosedBacklog}
	default:
		return Limits{New: ClosedNothing, Reviews: ClosedNothing, Minutes: ClosedMinutes, Backlog: ClosedBacklog}
	}
}
