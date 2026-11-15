package projection

import (
	"slices"
	"time"
)

// Card is where the answers so far have left one review card. Phase is the scheduler's own state: a
// simulation carries it from answer to answer and never reads it.
type Card struct {
	Due        time.Time
	Last       time.Time
	Reps       int
	Lapses     int
	Stability  float64
	Difficulty float64
	Phase      uint8
}

// IsNew reports whether the card has never been answered.
func (c Card) IsNew() bool { return c.Last.IsZero() }

func (c Card) isDue(d Calendar, at time.Time) bool { return c.IsNew() || c.Due.Before(d.GetEnd(at)) }

// Scheduler is what a simulation asks of the rules that schedule review cards.
type Scheduler interface {
	// GetEndings returns where a recalled and a forgotten answer given at this instant leave the card.
	GetEndings(c Card, at time.Time) (recalled, forgotten Card)
	// GetRecall returns the chance the card is recalled at this instant.
	GetRecall(c Card, at time.Time) float64
	// IsLearned reports whether the card counts as learned at this instant under these settings.
	IsLearned(s Settings, c Card, at time.Time) bool
	// GetAllowance returns what the review day holding open admits under these settings, after spent.
	GetAllowance(s Settings, open time.Time, spent Spent, unbegunCards, daysToLearn int) Allowance
	// NewLoad opens an empty count of the cards due on each day.
	NewLoad() Load
}

// Load counts the cards due on each review day.
type Load interface {
	// Add counts one card against the day its due time falls in.
	Add(due time.Time)
	// GetDayNumber returns the review day holding this instant as a whole number.
	GetDayNumber(at time.Time) int
	// PlaceCard returns the day a card answered at this instant is due under these settings, given the
	// scheduler's due time, and counts the card against that day.
	PlaceCard(s Settings, at, due time.Time) time.Time
}

// Calendar is where one review day ends and the next begins.
type Calendar interface {
	GetStart(at time.Time) time.Time
	GetEnd(at time.Time) time.Time
	GetName(at time.Time) string
	// GetEndOfDate returns the instant the review day of this calendar date ends.
	GetEndOfDate(date time.Time) time.Time
}

// RecallChance is the chance a simulation assumes that a card is recalled at an instant. Every figure
// of a run depends on it.
type RecallChance func(Card, time.Time) float64

// GetFullRecall is the chance under which every card is recalled.
func GetFullRecall(Card, time.Time) float64 { return 1 }

// Cost is how long one answer takes: for a card still being learned, and for a card that returns
// after days.
type Cost struct {
	New    time.Duration
	Review time.Duration
}

// Spent is what a review day has already used: cards answered, of them new and reviews, and time.
type Spent struct {
	Answered int
	New      int
	Reviews  int
	Took     time.Duration
}

// Allowance is what one review day admits: cards of each kind, time, the budget that closes each,
// the percentage of the day that goes to cards already begun, and whether the day schedules nothing.
type Allowance struct {
	New      int
	Reviews  int
	Minutes  time.Duration
	Limits   Limits
	Backlog  int
	IsPaused bool
}

// Limits is the budget that closes the day for new cards, for reviews and for time, and
// ClosedNothing where none does.
type Limits struct {
	New     BudgetName
	Reviews BudgetName
	Minutes BudgetName
}

// WholeBacklog is a day that offers every card already begun before a new one.
const WholeBacklog = 100

func (a Allowance) isNextCardDebt(debt, begun int, isDue, isNew bool) bool {
	if !isDue {
		return false
	}
	if !isNew {
		return true
	}
	return debt*WholeBacklog < a.Backlog*(debt+begun+1)
}

// BudgetName is a budget that closes a review day.
type BudgetName string

const (
	ClosedNothing BudgetName = ""
	ClosedMinutes BudgetName = "minutes_a_day"
	ClosedNew     BudgetName = "new_a_day"
	ClosedReviews BudgetName = "reviews_a_day"
	ClosedDate    BudgetName = "by_date"
	ClosedPaused  BudgetName = "paused"
)

// BudgetNames is every budget that closed one review day, in the order getClosers lists them.
type BudgetNames []BudgetName

func getClosers() []BudgetName {
	return []BudgetName{ClosedPaused, ClosedMinutes, ClosedNew, ClosedReviews, ClosedDate}
}

// Has reports whether this budget is one of those that closed the day.
func (c BudgetNames) Has(one BudgetName) bool { return slices.Contains(c, one) }

// Add returns these budgets and one more, each named once.
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

// Bounds is how far a value goes at each end.
type Bounds struct{ Min, Max float64 }
