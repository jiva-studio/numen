package providerbridge

import (
	"errors"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

func TestEveryFailureKindOfTheSchemaIsReadAsAFailure(t *testing.T) {
	testsupport.CheckHandled(t, func(kind v1.ProviderFailureKind) bool {
		err := readFailure(&v1.ProviderFailure{Kind: kind})
		var tryNext *connectionport.TryNextError
		var stopError *connectionport.StopError
		return errors.As(err, &tryNext) || errors.As(err, &stopError)
	})
}

func TestEveryReasonOfTheSchemaIsReadAsItsReason(t *testing.T) {
	expected := map[v1.ProviderFailureReason]connectionport.FailureReason{
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED:     connectionport.FailureReasonKeyRejected,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED: connectionport.FailureReasonPaymentRequired,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED:     connectionport.FailureReasonRateLimited,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE:      connectionport.FailureReasonUnreachable,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT:          connectionport.FailureReasonTimeout,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST:      connectionport.FailureReasonBadRequest,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN:          connectionport.FailureReasonUnknown,
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
