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

func TestEveryFailureReasonOfTheSchemaIsProducedFromTheCodeOfAFailure(t *testing.T) {
	testsupport.CheckProduced(t, map[v1.FailureReason]connection.FailureReason{
		v1.FailureReason_FAILURE_REASON_UNAVAILABLE:      connection.FailureReasonUnavailable,
		v1.FailureReason_FAILURE_REASON_REJECTED_KEY:     connection.FailureReasonRejectedKey,
		v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED: connection.FailureReasonPaymentRequired,
		v1.FailureReason_FAILURE_REASON_RATE_LIMITED:     connection.FailureReasonRateLimited,
		v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:  connection.FailureReasonInvalidRequest,
	}, func(reason connection.FailureReason) v1.FailureReason {
		served, _ := ConvertCallFailureToWire(t.Context(), &connection.AllFailedError{
			Failures: []connection.Failure{{ConnectionName: "Harbor", Reason: "words of the reason", Code: reason}},
		})
		return served.GetFailures()[0].GetCode()
	})
}

func TestTheFailureOfAStoppedCallCarriesTheCodeOfTheReason(t *testing.T) {
	served, _ := ConvertCallFailureToWire(t.Context(), &connection.StoppedError{
		ConnectionName: "Harbor", Reason: "invalid request", Code: connection.FailureReasonInvalidRequest,
	})
	if code := served.GetFailures()[0].GetCode(); code != v1.FailureReason_FAILURE_REASON_INVALID_REQUEST {
		t.Errorf("code = %v, want FAILURE_REASON_INVALID_REQUEST", code)
	}
}
