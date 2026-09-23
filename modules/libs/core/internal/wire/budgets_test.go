package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
)

// The budgets are a closed vocabulary the schema carries, so a window is told
// which budget closed a day and never reads the words a preset file writes.
// This is the whole of it: a value added to the schema is answered here before
// it can travel, and the window's own table is keyed by the schema, so it is
// asked for a word at the same moment.
func TestEveryBudgetTheSchemaNamesIsOneTheCoreHolds(t *testing.T) {
	held := map[v1.BudgetName]review.BudgetName{
		v1.BudgetName_BUDGET_NAME_UNSPECIFIED:   review.ClosedNothing,
		v1.BudgetName_BUDGET_NAME_MINUTES_A_DAY: review.ClosedMinutes,
		v1.BudgetName_BUDGET_NAME_NEW_A_DAY:     review.ClosedNew,
		v1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY: review.ClosedReviews,
		v1.BudgetName_BUDGET_NAME_BY_DATE:       review.ClosedDate,
		v1.BudgetName_BUDGET_NAME_BACKLOG:       review.ClosedBacklog,
		v1.BudgetName_BUDGET_NAME_PAUSED:        review.ClosedPaused,
	}

	for value := range v1.BudgetName_name {
		named := v1.BudgetName(value)
		one, answered := held[named]
		if !answered {
			t.Errorf("the schema names %s and nothing in the core answers to it", named)
			continue
		}
		if got := BudgetOf(one); got != named {
			t.Errorf("%q is sent as %s, want %s", one, got, named)
		}
	}
}

func TestTheBudgetsThatClosedADayTravelInTheOrderTheyWereNamed(t *testing.T) {
	got := BudgetsOf(review.BudgetNames{review.ClosedNew, review.ClosedReviews})
	want := []v1.BudgetName{
		v1.BudgetName_BUDGET_NAME_NEW_A_DAY,
		v1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY,
	}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the day was closed by %v, want %v", got, want)
	}
}
