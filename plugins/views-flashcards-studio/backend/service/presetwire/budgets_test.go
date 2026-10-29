package presetwire

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Every budget the schema names has a counterpart in the core. The conversion is
// keyed by the budget that closed a day and does not read the words a preset file
// contains. A value added to the schema fails here before it can be sent, and the
// window's table, which is keyed by the schema, is checked at the same time.
func TestEveryBudgetTheSchemaNamesIsOneTheCoreContains(t *testing.T) {
	target := map[api.BudgetName]review.BudgetName{
		api.BudgetNameUnspecified: review.ClosedNothing,
		api.BudgetNameMinutesADay: review.ClosedMinutes,
		api.BudgetNameNewADay:     review.ClosedNew,
		api.BudgetNameReviewsADay: review.ClosedReviews,
		api.BudgetNameByDate:      review.ClosedDate,
		api.BudgetNameBacklog:     review.ClosedBacklog,
		api.BudgetNamePaused:      review.ClosedPaused,
	}

	for _, named := range api.BudgetNameValues {
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
	want := []api.BudgetName{
		api.BudgetNameNewADay,
		api.BudgetNameReviewsADay,
	}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the day was closed by %v, want %v", got, want)
	}
}
