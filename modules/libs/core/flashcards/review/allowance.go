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

// on is the budget a preset keeps on this day of the week: its share of the
// load, whether or not the days are evened out.
//
// What a day of it admits is Admits, which is the one place a limit is read.
func (p Preset) on(day time.Weekday) Budget {
	share := p.Share(day)
	return Budget{
		New:     int(math.Round(share * float64(p.NewADay))),
		Reviews: int(math.Round(share * float64(p.ReviewsADay))),
		Minutes: share * float64(p.MinutesADay),
	}
}

// BudgetName is one budget a day of review may be stopped by, in the words the
// preset writes the key in. A budget the goal does not name is never one of
// them, and a day that asked for everything there was is closed by nothing:
// the material ran out.
type BudgetName string

const (
	ClosedNothing BudgetName = ""
	ClosedMinutes BudgetName = "minutes_a_day"
	ClosedNew     BudgetName = "new_a_day"
	ClosedReviews BudgetName = "reviews_a_day"
	// ClosedDate is a day paced by the day the preset aims at.
	ClosedDate BudgetName = "by_date"
	// ClosedBacklog is the share of a day that goes to the debt. It closes
	// nothing, and stands in this vocabulary because it is a key of a preset
	// that a goal either reads or leaves idle.
	ClosedBacklog BudgetName = "backlog"
	// ClosedPaused is a preset scheduling nothing at all.
	ClosedPaused BudgetName = "paused"
)

// BudgetNames is every budget that closed one day of review.
//
// A goal of retention holds a day to both card counts, and a day that ran out
// of new cards and of reviews names both: a person raising one of them and
// finding nothing changed is reading a day the other closed too.
type BudgetNames []BudgetName

// closers is every budget a day may be closed by, in the order a closing names
// them.
var closers = []BudgetName{ClosedPaused, ClosedMinutes, ClosedNew, ClosedReviews, ClosedDate}

// Holds reports whether this budget is one of those that closed the day.
func (c BudgetNames) Holds(one BudgetName) bool { return slices.Contains(c, one) }

// with is this closing and one budget more, named once and in the order closers
// stands in.
func (c BudgetNames) with(one BudgetName) BudgetNames {
	if !slices.Contains(closers, one) || c.Holds(one) {
		return c
	}
	out := make(BudgetNames, 0, len(c)+1)
	for _, each := range closers {
		if each == one || c.Holds(each) {
			out = append(out, each)
		}
	}
	return out
}

// Name is the key one budget is written under, and empty where the day was
// closed by none or by more than one.
func (c BudgetNames) Name() string {
	if len(c) != 1 {
		return ""
	}
	return string(c[0])
}

// Limits is which key caps each budget under the goal in force, each written as
// the preset writes the key, and empty where it takes no part.
//
// A setting taking no part stands in the file where the person left it and is
// in force again the moment its goal is chosen.
type Limits struct {
	// New, Reviews and Minutes are the budgets. The goal names the one that
	// closes the day, and the others take no part.
	New     BudgetName
	Reviews BudgetName
	Minutes BudgetName
	// Backlog is the share of the day that goes to the debt. It closes nothing:
	// it says what the day is spent on, and it is empty under a goal whose day
	// is not one pot spent between the two. A goal of a date carries the whole
	// material by its own reckoning, and a goal of retention holds each side to
	// a count of its own.
	Backlog BudgetName
}

// Allowance is what one day of a preset admits: how many cards of each kind it
// has room for, how long the day still runs, and which of the three closes it.
//
// It is the one answer to what a day admits: a session spends against it, and a
// projection runs on it.
type Allowance struct {
	// Keeps is what the preset keeps for the whole of this day, the day of the
	// week having had its say.
	Keeps Budget
	// New, Reviews and Minutes are what is left of it.
	New     int
	Reviews int
	Minutes time.Duration
	// Limits is which of the three closes the day.
	Limits Limits
	// Backlog is how much of the day goes to the debt before anything unbegun
	// is offered, as a share in hundredths. A goal whose day is not one pot
	// spent between the two stands at the whole of it, and the debt is paid
	// first.
	Backlog int
	// Stops is why this day schedules nothing, and empty where it schedules
	// something.
	Stops StopReason
}

// IsPaused reports whether this day schedules nothing.
func (a Allowance) IsPaused() bool { return a.Stops != StoppedNothing }

// Admits is what this preset's day admits.
//
// Now is any instant of the review day, spent is what that day has already gone
// through under the preset, unbegunCards is the material it has still to begin,
// and daysToLearn is how many days of review a card face begun now needs before
// the preset counts it learned, which a date paces the day against.
func (p Preset) Admits(
	d Day, now time.Time, spent Spent, unbegunCards, daysToLearn int,
) Allowance {
	opened := d.GetDate(now)
	out := Allowance{
		Keeps:  p.on(opened.Weekday()),
		Limits: p.limits(),
		Stops:  p.StopsOn(d, now),
	}
	if p.Goal == GoalDate {
		// The pace is a whole day's share of the material, and this day carries
		// as much of it as its day of the week carries of the load.
		out.Keeps.New = int(math.Round(
			p.Share(opened.Weekday()) * float64(p.paces(d, now, unbegunCards, daysToLearn))))
	}
	// A split that takes no part leaves the day spending on the debt first,
	// which is what a preset naming no share does.
	out.Backlog = AllBacklog
	if out.Limits.Backlog != ClosedNothing {
		out.Backlog = p.Backlog
	}
	out.New = out.Keeps.New - spent.New
	out.Reviews = out.Keeps.Reviews - spent.Reviews
	out.Minutes = time.Duration(out.Keeps.Minutes*float64(time.Minute)) - spent.Took
	return out
}

// Paying reports whether the next card of this day comes from the debt before
// it. False is a card from the material the preset has not begun.
//
// Debt and begun are how many of each the day has taken so far, and owed and
// fresh whether either side has a card left to give. The day is spent between
// the two in the share the preset names; a side with nothing left leaves the
// rest of the day to the other, and an odd card goes to the debt.
//
// It is the one rule for how a day is spent.
func (a Allowance) IsPayingDebt(debt, begun int, owed, fresh bool) bool {
	if !owed {
		return false
	}
	if !fresh {
		return true
	}
	return debt*AllBacklog < a.Backlog*(debt+begun+1)
}

// limits is which budget closes this preset's day.
//
// A goal of a date closes the day on a count of new cards, which is the share
// of the material a day has to begin to be through it by then. The date is what
// paced that count, so the date is what closed the day.
//
// The share of the day that goes to the debt is read where one pot is spent
// between the two. A goal of retention keeps a count for each side, so each is
// held to its own and the share decides nothing.
func (p Preset) limits() Limits {
	switch p.Goal {
	case GoalRetention:
		return Limits{New: ClosedNew, Reviews: ClosedReviews}
	case GoalDate:
		return Limits{New: ClosedDate}
	default:
		return Limits{Minutes: ClosedMinutes, Backlog: ClosedBacklog}
	}
}
