package settings

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// A read does not wait for a provider that is slow to list its programs: it returns the programs
// listed last, and a later read returns what the slow listing found.
func TestAReadDoesNotWaitForASlowListingOfPrograms(t *testing.T) {
	var programs ProgramList
	release := make(chan struct{})
	slow := func() []agentport.AgentProgram {
		<-release
		return []agentport.AgentProgram{{Name: "claude"}}
	}

	first := programs.Get(20*time.Millisecond, slow)
	if len(first) != 0 {
		t.Errorf("the first read returned %v, want no programs before a listing ended", first)
	}
	during := programs.Get(20*time.Millisecond, slow)
	if len(during) != 0 {
		t.Errorf("a read during the listing returned %v, want the programs listed last", during)
	}

	close(release)
	for range 500 {
		if got := programs.Get(time.Second, slow); len(got) == 1 && got[0].Name == "claude" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("the programs of the slow listing never reached a read")
}

// A listing that ends within the wait is what the read returns.
func TestAReadReturnsTheProgramsOfAFastListing(t *testing.T) {
	var programs ProgramList
	got := programs.Get(time.Second, func() []agentport.AgentProgram {
		return []agentport.AgentProgram{{Name: "codex"}}
	})
	if len(got) != 1 || got[0].Name != "codex" {
		t.Errorf("read = %v, want codex", got)
	}
}
