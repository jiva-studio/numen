package connection

import (
	"reflect"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/schematest"
)

func TestACallIsReadFromTheWire(t *testing.T) {
	got := readCall(&v1.ProviderCall{
		Type:      "openrouter",
		Fields:    map[string]string{"baseUrl": "https://openrouter.ai/api/v1"},
		Secrets:   []*v1.ProviderSecret{{Key: "apiKey", Value: "sk-1"}},
		Model:     "openai/gpt-5",
		Reasoning: "low",
	})
	want := Call{
		Type:      "openrouter",
		Fields:    map[string]string{"baseUrl": "https://openrouter.ai/api/v1"},
		Secrets:   map[string]Secret{"apiKey": {Value: "sk-1"}},
		Model:     "openai/gpt-5",
		Reasoning: "low",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readCall = %#v, want %#v", got, want)
	}
}

func TestANilWireCallReadsAsAnEmptyCall(t *testing.T) {
	got := readCall(nil)
	if got.Type != "" || got.Model != "" || len(got.Fields) != 0 || len(got.Secrets) != 0 {
		t.Errorf("readCall(nil) = %#v, want the empty call", got)
	}
}

func TestEveryReasonOfTheSchemaIsEncoded(t *testing.T) {
	schematest.CheckProduced(t, map[v1.ProviderFailureReason]Reason{
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_KEY_REJECTED:     ReasonKeyRejected,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_PAYMENT_REQUIRED: ReasonPaymentRequired,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED:     ReasonRateLimited,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNREACHABLE:      ReasonUnreachable,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_TIMEOUT:          ReasonTimeout,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_BAD_REQUEST:      ReasonBadRequest,
		v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN:          ReasonUnknown,
	}, writeReason)
}

func TestEveryFailureKindOfTheSchemaIsEncoded(t *testing.T) {
	schematest.CheckProduced(t, map[v1.ProviderFailureKind]error{
		v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT: NewTryNextError(ReasonTimeout, 0),
		v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP:     NewStopError(ReasonBadRequest),
	}, func(err error) v1.ProviderFailureKind { return writeFailure(err).GetKind() })
}

func TestAFailureIsEncodedWithItsKindReasonAndRetryAfter(t *testing.T) {
	wire := writeFailure(NewTryNextError(ReasonRateLimited, 1500*time.Millisecond))
	if wire.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_TRY_NEXT ||
		wire.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_RATE_LIMITED || wire.GetRetryAfterMs() != 1500 {
		t.Fatalf("wire = %v", wire)
	}
}

func TestAPlainErrorIsEncodedAsAStopOfUnknownReason(t *testing.T) {
	wire := writeFailure(errPlain)
	if wire.GetKind() != v1.ProviderFailureKind_PROVIDER_FAILURE_KIND_STOP ||
		wire.GetReason() != v1.ProviderFailureReason_PROVIDER_FAILURE_REASON_UNKNOWN {
		t.Errorf("wire = %v, want a stop of unknown reason", wire)
	}
}

func TestANegativeRetryAfterIsEncodedAsNone(t *testing.T) {
	if got := writeFailure(NewTryNextError(ReasonTimeout, -time.Second)).GetRetryAfterMs(); got != 0 {
		t.Errorf("retry_after_ms = %d, want 0", got)
	}
}
