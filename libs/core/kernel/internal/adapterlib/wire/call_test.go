package wire

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestEveryErrorCodeOfTheSchemaIsProducedFromAFailure(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.CallErrorCode]error{
		v1.CallErrorCode_CALL_ERROR_CODE_NO_CONNECTION: connection.ErrNoConnection,
		v1.CallErrorCode_CALL_ERROR_CODE_ALL_FAILED:    &connection.AllFailedError{},
		v1.CallErrorCode_CALL_ERROR_CODE_STOPPED:       &connection.StoppedError{},
	}, func(failure error) v1.CallErrorCode {
		served, _ := ConvertCallFailureToWire(t.Context(), failure)
		return served.GetError()
	})
}

func TestEveryPriorityOfTheSchemaIsReadAsThePortOne(t *testing.T) {
	want := map[v1.Priority]modelcall.Priority{
		v1.Priority_PRIORITY_INTERACTIVE: modelcall.PriorityInteractive,
		v1.Priority_PRIORITY_BACKGROUND:  modelcall.PriorityBackground,
	}
	testsupport.CheckHandled(t, func(priority v1.Priority) bool { return ConvertPriorityFromWire(priority) == want[priority] })
}
