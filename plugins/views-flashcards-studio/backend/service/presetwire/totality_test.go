package presetwire

import (
	"testing"

	flashcardsv1 "github.com/jiva-studio/numen/plugins/views-flashcards-studio/gen/numen/flashcards/v1"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/schematest"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

// Every value of every enum this package carries across reaches something.
//
// The values are walked from the schema, so a value added there is a test that fails and not a
// message that quietly says nothing.

func TestEveryGoalCrossesBothWays(t *testing.T) {
	schematest.RoundTrip(t, ConvertGoalFromWire, ConvertGoalToWire)
}

func TestEveryRuleCrossesBothWays(t *testing.T) {
	schematest.RoundTrip(t, ConvertRuleFromWire, ConvertRuleToWire)
}

func TestEveryBudgetUnitCrossesBothWays(t *testing.T) {
	schematest.RoundTrip(t, ConvertBudgetUnitFromWire, ConvertBudgetUnitToWire)
}

func TestEveryVerdictIsSavedFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[flashcardsv1.StopReason]review.StopReason{
		flashcardsv1.StopReason_STOP_REASON_NOTHING:    review.StoppedNothing,
		flashcardsv1.StopReason_STOP_REASON_NO_MINUTES: review.StoppedNoMinutes,
		flashcardsv1.StopReason_STOP_REASON_NO_CARDS:   review.StoppedNoCards,
		flashcardsv1.StopReason_STOP_REASON_NO_DAY:     review.StoppedNoDay,
		flashcardsv1.StopReason_STOP_REASON_PAST_DAY:   review.StoppedPastDay,
		flashcardsv1.StopReason_STOP_REASON_NO_LOAD:    review.StoppedNoLoad,
		flashcardsv1.StopReason_STOP_REASON_NO_WEEK:    review.StoppedNoWeek,
	}, ConvertStopReasonToWire)
}
