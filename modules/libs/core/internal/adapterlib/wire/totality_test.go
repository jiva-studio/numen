package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

// Every value of every enum this file carries across reaches something.
//
// The values are walked from the schema, so a value added there is a test that
// fails and not a message that quietly says nothing.

func TestEveryModeCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, ConvertModeFromWire, ConvertModeToWire)
}

func TestEveryUnitIsWrittenFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Unit]run.Unit{
		v1.Unit_UNIT_THINGS:  run.Things,
		v1.Unit_UNIT_BYTES:   run.Bytes,
		v1.Unit_UNIT_SECONDS: run.Seconds,
	}, convertUnit)
}

// A page that named nothing has said nothing, and silence is what the round
// waits its bound for. Every answer the schema offers is one of the answers.
func TestEveryFlushResultIsAnAnswer(t *testing.T) {
	testsupport.CheckHandled(t, func(said v1.FlushResult) bool { return parseFlushResult(said) != flushUnanswered })
}
