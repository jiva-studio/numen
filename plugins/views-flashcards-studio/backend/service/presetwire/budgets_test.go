package presetwire

import (
	"testing"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// which budget closed a day and never reads the words a preset file writes.
// This is the whole of it: a value added to the schema is answered here before
// it can travel, and the window's own table is keyed by the schema, so it is
// asked for a word at the same moment.
func TestEveryBudgetTheSchemaNamesIsOneTheCoreContains(t *testing.T) {
	target := map[flashcardsv1.BudgetName]review.BudgetName{
		flashcardsv1.BudgetName_BUDGET_NAME_UNSPECIFIED:   review.ClosedNothing,
		flashcardsv1.BudgetName_BUDGET_NAME_MINUTES_A_DAY: review.ClosedMinutes,
		flashcardsv1.BudgetName_BUDGET_NAME_NEW_A_DAY:     review.ClosedNew,
		flashcardsv1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY: review.ClosedReviews,
		flashcardsv1.BudgetName_BUDGET_NAME_BY_DATE:       review.ClosedDate,
		flashcardsv1.BudgetName_BUDGET_NAME_BACKLOG:       review.ClosedBacklog,
		flashcardsv1.BudgetName_BUDGET_NAME_PAUSED:        review.ClosedPaused,
	}

	for value := range flashcardsv1.BudgetName_name {
		named := flashcardsv1.BudgetName(value)
		one, reply := target[named]
		if !reply {
			t.Errorf("the schema names %s and nothing in the core answers to it", named)
			continue
		}
		if got := ConvertBudgetToWire(one); got != named {
			t.Errorf("%q is sent as %s, want %s", one, got, named)
		}
	}
}

func TestTheBudgetsThatClosedADayTravelInTheOrderTheyWereNamed(t *testing.T) {
	got := ConvertBudgetsToWire(review.BudgetNames{review.ClosedNew, review.ClosedReviews})
	want := []flashcardsv1.BudgetName{
		flashcardsv1.BudgetName_BUDGET_NAME_NEW_A_DAY,
		flashcardsv1.BudgetName_BUDGET_NAME_REVIEWS_A_DAY,
	}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the day was closed by %v, want %v", got, want)
	}
}
