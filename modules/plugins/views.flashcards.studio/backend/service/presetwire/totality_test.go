package presetwire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/schematest"

	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/review"
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

func TestEveryVerdictIsWrittenFromOne(t *testing.T) {
	schematest.CheckProduced(t, map[v1.StopReason]review.StopReason{
		v1.StopReason_STOP_REASON_NOTHING:    review.StoppedNothing,
		v1.StopReason_STOP_REASON_NO_MINUTES: review.StoppedNoMinutes,
		v1.StopReason_STOP_REASON_NO_CARDS:   review.StoppedNoCards,
		v1.StopReason_STOP_REASON_NO_DAY:     review.StoppedNoDay,
		v1.StopReason_STOP_REASON_PAST_DAY:   review.StoppedPastDay,
		v1.StopReason_STOP_REASON_NO_LOAD:    review.StoppedNoLoad,
		v1.StopReason_STOP_REASON_NO_WEEK:    review.StoppedNoWeek,
	}, ConvertStopReasonToWire)
}
