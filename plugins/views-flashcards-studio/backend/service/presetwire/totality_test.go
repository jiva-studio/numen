package presetwire

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/enumtest"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Every value of every enum this package carries across reaches something.
//
// The values are walked from the schema, so a value added there is a test that fails and not a
// message that quietly says nothing.

func TestEveryGoalCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.GoalValues, ConvertGoalFromWire, ConvertGoalToWire)
}

func TestEveryRuleCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.RuleValues, ConvertRuleFromWire, ConvertRuleToWire)
}

func TestEveryBudgetUnitCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.BudgetUnitValues, ConvertBudgetUnitFromWire, ConvertBudgetUnitToWire)
}

func TestEveryVerdictIsSavedFromOne(t *testing.T) {
	enumtest.CheckProduced(t, api.StopReasonValues, map[api.StopReason]review.StopReason{
		api.StopReasonNothing:   review.StoppedNothing,
		api.StopReasonNoMinutes: review.StoppedNoMinutes,
		api.StopReasonNoCards:   review.StoppedNoCards,
		api.StopReasonNoDay:     review.StoppedNoDay,
		api.StopReasonPastDay:   review.StoppedPastDay,
		api.StopReasonNoLoad:    review.StoppedNoLoad,
		api.StopReasonNoWeek:    review.StoppedNoWeek,
	}, ConvertStopReasonToWire)
}
