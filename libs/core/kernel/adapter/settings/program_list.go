package settings

import (
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

// ProgramList keeps the programs the agent providers listed last, and lists them again in the
// background, so a read of the settings does not wait for a provider that is slow to start.
type ProgramList struct {
	mu       sync.Mutex
	programs []agent.Program
	late     []agent.Program
	isAsking bool
	isLate   bool
}

// Get starts a listing unless one is running, and returns what the listing found within wait, or
// else the programs listed before. When a listing ends after wait with different programs, each
// function in onLate is called.
func (l *ProgramList) Get(wait time.Duration, list func() []agent.Program, onLate ...func()) []agent.Program {
	ended := make(chan struct{})
	l.mu.Lock()
	if l.isAsking {
		close(ended)
	} else {
		l.isAsking = true
		go l.runList(ended, list, onLate)
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
	if l.isAsking {
		l.isLate = true
	}
	return slices.Clone(l.programs)
}

func (l *ProgramList) runList(ended chan<- struct{}, list func() []agent.Program, onLate []func()) {
	found := list()
	l.mu.Lock()
	notify := l.isLate && !reflect.DeepEqual(l.programs, found)
	if notify {
		l.late = slices.Clone(found)
	}
	l.programs, l.isAsking, l.isLate = found, false, false
	l.mu.Unlock()
	close(ended)
	if !notify {
		return
	}
	for _, fn := range onLate {
		if fn != nil {
			fn()
		}
	}
}

// GetLate returns the programs of the last listing that ended after Get timed out.
func (l *ProgramList) GetLate() []agent.Program {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.late)
}
