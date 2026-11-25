package tools

import (
	"testing"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// Every value the protocol defines for an enum maps to an SDK value, so a value
// added to the protocol without a mapping fails here.
func TestEnums_EveryProtocolValueRoundTrips(t *testing.T) {
	for value := range v1.ToolEffect_name {
		effect := v1.ToolEffect(value)
		if got := writeEffect(readEffect(effect)); got != effect {
			t.Errorf("effect %v came back as %v", effect, got)
		}
	}
	for _, code := range []v1.ErrorCode{
		v1.ErrorCode_ERROR_CODE_NOT_FOUND,
		v1.ErrorCode_ERROR_CODE_INVALID,
	} {
		if got := readErrorCode(code); got == ErrorCodeUnspecified {
			t.Errorf("error code %v reads as no error", code)
		}
	}
}
