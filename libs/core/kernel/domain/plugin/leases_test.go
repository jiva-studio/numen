package plugin_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func TestReturningALeaseTheHolderDoesNotHoldRemovesNone(t *testing.T) {
	var leases plugin.Leases
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	leases.Add(plugin.Holder{Who: plugin.WhoWindow, Why: "open view"}, true, now)

	isLast := leases.Remove(plugin.Holder{Who: plugin.WhoSearch, Why: "search notes"}, true, now)

	if isLast || leases.Count() != 1 {
		t.Errorf("after a release by another holder: last %v, %d leases, want the one lease kept", isLast, leases.Count())
	}
}
