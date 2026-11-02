package settings

import (
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// ProgramList keeps the programs the agent providers listed last, and lists them again in the
// background, so a read of the settings does not wait for a provider that is slow to start.
type ProgramList struct {
	mu       sync.Mutex
	programs []agentport.AgentProgram
	isAsking bool
}

// Get starts a listing unless one is running, and returns what the listing found within wait, or
// else the programs listed before.
func (l *ProgramList) Get(wait time.Duration, list func() []agentport.AgentProgram) []agentport.AgentProgram {
	ended := make(chan struct{})
	l.mu.Lock()
	if l.isAsking {
		close(ended)
	} else {
		l.isAsking = true
		go func() {
			found := list()
			l.mu.Lock()
			l.programs, l.isAsking = found, false
			l.mu.Unlock()
			close(ended)
		}()
	}
	l.mu.Unlock()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ended:
	case <-timer.C:
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.programs)
}
