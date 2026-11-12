package connectionlog

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

func TestLogNoticesWritesTheConnectionAndTheReason(t *testing.T) {
	var out strings.Builder

	NewLogNotices(&out).ReportSetAside("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired)

	for _, want := range []string{"OpenRouter", "c0000001", "payment required"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("log = %q, want it to contain %q", out.String(), want)
		}
	}
}
