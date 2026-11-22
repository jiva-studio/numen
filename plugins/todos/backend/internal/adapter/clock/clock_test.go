package clock_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/todos/backend/internal/adapter/clock"
)

func TestSystemGetTodayReturnsALocalDate(t *testing.T) {
	today := clock.System{}.GetToday()
	if _, err := time.Parse(time.DateOnly, today); err != nil {
		t.Fatalf("GetToday() = %q, want a YYYY-MM-DD date: %v", today, err)
	}
}
