package presetwire

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/enumtest"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/review"
)

// Every value of every enum this package converts maps to a value.
//
// The values are enumerated from the schema, so a value added there fails this
// test and does not convert to a message with no content.

func TestEveryGoalCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.GetGoalValues(), ConvertGoalFromWire, ConvertGoalToWire)
}

func TestEveryRuleCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.GetRuleValues(), ConvertRuleFromWire, ConvertRuleToWire)
}

func TestEveryBudgetUnitCrossesBothWays(t *testing.T) {
	enumtest.RoundTrip(t, api.GetBudgetUnitValues(), ConvertBudgetUnitFromWire, ConvertBudgetUnitToWire)
}

func TestEveryVerdictIsSavedFromOne(t *testing.T) {
	enumtest.CheckProduced(t, api.GetStopReasonValues(), map[api.StopReason]review.StopReason{
		api.StopReasonNothing:   review.StoppedNothing,
		api.StopReasonNoMinutes: review.StoppedNoMinutes,
		api.StopReasonNoCards:   review.StoppedNoCards,
		api.StopReasonNoDay:     review.StoppedNoDay,
		api.StopReasonPastDay:   review.StoppedPastDay,
		api.StopReasonNoLoad:    review.StoppedNoLoad,
		api.StopReasonNoWeek:    review.StoppedNoWeek,
	}, ConvertStopReasonToWire)
}
