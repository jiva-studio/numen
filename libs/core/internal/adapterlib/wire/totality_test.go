package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

// Every value of every enum this file converts maps to a defined result.
//
// The values are enumerated from the schema, so a value added there fails this
// test instead of being silently ignored.

func TestEveryModeCrossesBothWays(t *testing.T) {
	testsupport.RoundTrip(t, ConvertModeFromWire, ConvertModeToWire)
}

func TestEveryUnitIsSavedFromOne(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.Unit]run.Unit{
		v1.Unit_UNIT_THINGS:  run.Things,
		v1.Unit_UNIT_BYTES:   run.Bytes,
		v1.Unit_UNIT_SECONDS: run.Seconds,
	}, convertUnit)
}

// A page that reported nothing is unanswered, and the round waits its bound for
// unanswered pages. Every answer the schema defines counts as an answer.
func TestEveryFlushResultIsAnAnswer(t *testing.T) {
	testsupport.CheckHandled(t, func(response v1.FlushResult) bool { return parseFlushResult(response) != flushUnanswered })
}
