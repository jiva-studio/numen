package providerbridge

import (
	"errors"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestEveryFailureKindOfTheSchemaIsReadAsAFailure(t *testing.T) {
	testsupport.CheckHandled(t, func(kind v1.FailureKind) bool {
		err := readFailure(&v1.ProviderFailure{Kind: kind})
		var tryNext *connection.TryNextError
		var stopError *connection.StopError
		return errors.As(err, &tryNext) || errors.As(err, &stopError)
	})
}

func TestEveryReasonOfTheSchemaIsReadAsItsReason(t *testing.T) {
	expected := map[v1.FailureReason]connection.FailureReason{
		v1.FailureReason_FAILURE_REASON_REJECTED_KEY:     connection.FailureReasonRejectedKey,
		v1.FailureReason_FAILURE_REASON_PAYMENT_REQUIRED: connection.FailureReasonPaymentRequired,
		v1.FailureReason_FAILURE_REASON_RATE_LIMITED:     connection.FailureReasonRateLimited,
		v1.FailureReason_FAILURE_REASON_UNAVAILABLE:      connection.FailureReasonUnavailable,
		v1.FailureReason_FAILURE_REASON_INVALID_REQUEST:  connection.FailureReasonInvalidRequest,
	}
	for number := range v1.FailureReason_name {
		reason := v1.FailureReason(number)
		if reason == v1.FailureReason_FAILURE_REASON_UNSPECIFIED {
			continue
		}
		want, isNamed := expected[reason]
		if !isNamed {
			t.Errorf("%s is read as nothing", reason)
		} else if got := readReason(reason); got != want {
			t.Errorf("%s is read as %v, want %v", reason, got, want)
		}
	}
}
