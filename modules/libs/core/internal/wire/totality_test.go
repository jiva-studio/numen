package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/task"
)

// Every value of every enum this file carries across reaches something.
//
// The values are walked from the schema, so a value added there is a test that
// fails and not a message that quietly says nothing.

func TestEveryModeCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, ModeIn, ModeOf)
}

func TestEveryGoalCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, GoalIn, GoalOf)
}

func TestEveryRuleCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, RuleIn, RuleOf)
}

func TestEveryBudgetUnitCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, BudgetUnitIn, BudgetUnitOf)
}

func TestEveryVerdictIsWrittenFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.StopReason]review.StopReason{
		v1.StopReason_STOP_REASON_NOTHING:    review.StoppedNothing,
		v1.StopReason_STOP_REASON_NO_MINUTES: review.StoppedNoMinutes,
		v1.StopReason_STOP_REASON_NO_CARDS:   review.StoppedNoCards,
		v1.StopReason_STOP_REASON_NO_DAY:     review.StoppedNoDay,
		v1.StopReason_STOP_REASON_PAST_DAY:   review.StoppedPastDay,
		v1.StopReason_STOP_REASON_NO_LOAD:    review.StoppedNoLoad,
		v1.StopReason_STOP_REASON_NO_WEEK:    review.StoppedNoWeek,
	}, StopReasonOf)
}

func TestEveryUnitIsWrittenFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Unit]task.Unit{
		v1.Unit_UNIT_THINGS:  task.Things,
		v1.Unit_UNIT_BYTES:   task.Bytes,
		v1.Unit_UNIT_SECONDS: task.Seconds,
	}, unitOf)
}

// A page that named nothing has said nothing, and silence is what the round
// waits its bound for. Every answer the schema offers is one of the answers.
func TestEveryFlushResultIsAnAnswer(t *testing.T) {
	testsupport.CheckHandled(t, func(said v1.FlushResult) bool { return left(said) != silent })
}
