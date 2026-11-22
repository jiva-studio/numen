package clock_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/clock"
)

func TestSystemGetTodayReturnsALocalDate(t *testing.T) {
	today := clock.System{}.Now().Format(time.DateOnly)
	if _, err := time.Parse(time.DateOnly, today); err != nil {
		t.Fatalf("Now() = %q, want a YYYY-MM-DD date: %v", today, err)
	}
}
