package loops_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/loops"
)

func TestInterval_TooLongForDurationUsesFallback(t *testing.T) {
	interval := loops.NewInterval(30)

	interval.Set("9223372037")

	if got := interval.Get(); got != 30*time.Second {
		t.Errorf("the interval is %s, want the fallback of 30s", got)
	}
}

func TestInterval_OfSecondsIsUsed(t *testing.T) {
	interval := loops.NewInterval(30)

	interval.Set("90")

	if got := interval.Get(); got != 90*time.Second {
		t.Errorf("the interval is %s, want 1m30s", got)
	}
}
