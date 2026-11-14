package refresh

import (
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// timers hold, for each vault, the timer that runs its due searches when the earliest cooldown of a stale
// search ends, while the process lives. No lease is held for it.
type timers struct {
	mu     sync.Mutex
	timers map[domain.VaultID]*time.Timer
}

func newTimers() *timers {
	return &timers{mu: sync.Mutex{}, timers: map[domain.VaultID]*time.Timer{}}
}

func (t *timers) set(vault domain.VaultID, delay time.Duration, run func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timer, isSet := t.timers[vault]; isSet {
		timer.Stop()
		delete(t.timers, vault)
	}
	if delay > 0 {
		t.timers[vault] = time.AfterFunc(delay, run)
	}
}
