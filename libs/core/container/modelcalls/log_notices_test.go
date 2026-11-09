package modelcalls

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/port/connectionport"
)

func TestLogNoticesWritesTheConnectionAndTheReason(t *testing.T) {
	var out strings.Builder

	NewLogNotices(&out).ReportSetAside("c0000001", "OpenRouter", connectionport.FailureReasonPaymentRequired)

	for _, want := range []string{"OpenRouter", "c0000001", "payment required"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("log = %q, want it to contain %q", out.String(), want)
		}
	}
}
