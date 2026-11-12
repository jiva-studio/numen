package settings

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

// A read does not wait for a provider that is slow to list its programs: it returns the programs
// listed last, notifies when the slow listing ends, and returns what the slow listing found on a
// later read.
func TestAReadDoesNotWaitForASlowListingOfPrograms(t *testing.T) {
	var programs ProgramList
	release := make(chan struct{})
	notice := make(chan struct{})
	slow := func() []agent.Program {
		<-release
		return []agent.Program{{Name: "claude"}}
	}

	first := programs.Get(20*time.Millisecond, slow, func() { close(notice) })
	if len(first) != 0 {
		t.Errorf("the first read returned %v, want no programs before a listing ended", first)
	}
	during := programs.Get(20*time.Millisecond, slow)
	if len(during) != 0 {
		t.Errorf("a read during the listing returned %v, want the programs listed last", during)
	}

	close(release)
	select {
	case <-notice:
	case <-time.After(2 * time.Second):
		t.Fatal("the slow listing ended without notifying")
	}
	if late := programs.GetLate(); len(late) != 1 || late[0].Name != "claude" {
		t.Errorf("GetLate() = %v, want claude", late)
	}
	if got := programs.Get(time.Second, slow); len(got) != 1 || got[0].Name != "claude" {
		t.Errorf("read after slow listing = %v, want claude", got)
	}
}

// A listing that ends within the wait is what the read returns, without a late notification.
func TestAReadReturnsTheProgramsOfAFastListing(t *testing.T) {
	var programs ProgramList
	isNotified := false
	got := programs.Get(time.Second, func() []agent.Program {
		return []agent.Program{{Name: "codex"}}
	}, func() {
		isNotified = true
	})
	if len(got) != 1 || got[0].Name != "codex" {
		t.Errorf("read = %v, want codex", got)
	}
	if isNotified {
		t.Error("a fast listing notified as late")
	}
}
