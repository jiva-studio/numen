package connectionlog

import (
	"os"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

func TestLogNoticesWritesTheConnectionAndTheReason(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})

	LogNotices{out: file}.ReportSetAside("c0000001", "OpenRouter", connection.FailureReasonPaymentRequired)

	written, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"OpenRouter", "c0000001", "payment required"} {
		if !strings.Contains(string(written), want) {
			t.Errorf("log = %q, want it to contain %q", written, want)
		}
	}
}
