package providerbridge

import (
	"errors"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestEveryFailureKindOfTheSchemaIsReadAsAFailure(t *testing.T) {
	testsupport.CheckHandled(t, func(kind v1.ProviderFailureKind) bool {
		err := readFailure(&v1.ProviderFailure{Kind: kind})
		var tryNext *connection.TryNextError
		var stopError *connection.StopError
		return errors.As(err, &tryNext) || errors.As(err, &stopError)
	})
}

func TestEveryReasonOfTheSchemaIsReadAsItsReason(t *testing.T) {
	expected := map[v1.ProviderFailureReason]connection.FailureReason{
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED:     connection.FailureReasonRejectedKey,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED: connection.FailureReasonPaymentRequired,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED:     connection.FailureReasonRateLimited,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE:      connection.FailureReasonUnavailable,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT:          connection.FailureReasonUnavailable,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST:      connection.FailureReasonInvalidRequest,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN:          connection.FailureReasonUnavailable,
	}
	for number := range v1.ProviderFailureReason_name {
		reason := v1.ProviderFailureReason(number)
		if reason == v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNSPECIFIED {
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
