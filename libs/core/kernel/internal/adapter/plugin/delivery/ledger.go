package delivery

import "sync"

// ledger remembers, for each plugin, whether it was given its configuration, whether it refused it, and
// which plugin has not been configured since it started.
type ledger struct {
	mu                     sync.RWMutex
	delivered              map[string]bool
	rejected               map[string]bool
	awaitingFirstConfigure map[string]bool
}

func newLedger() *ledger {
	return &ledger{
		delivered:              make(map[string]bool),
		rejected:               make(map[string]bool),
		awaitingFirstConfigure: make(map[string]bool),
	}
}

func (l *ledger) markAwaitingFirstConfigure(pluginID string) {
	l.mu.Lock()
	delete(l.delivered, pluginID)
	delete(l.rejected, pluginID)
	l.awaitingFirstConfigure[pluginID] = true
	l.mu.Unlock()
}

func (l *ledger) clearAwaitingFirstConfigure(pluginID string) {
	l.mu.Lock()
	delete(l.awaitingFirstConfigure, pluginID)
	l.mu.Unlock()
}

func (l *ledger) isAwaitingFirstConfigure(pluginID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.awaitingFirstConfigure[pluginID]
}

func (l *ledger) forget(pluginIDs []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, id := range pluginIDs {
		delete(l.delivered, id)
		delete(l.rejected, id)
	}
}

func (l *ledger) isDue(pluginID string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.awaitingFirstConfigure[pluginID] && !l.delivered[pluginID] && !l.rejected[pluginID]
}

func (l *ledger) recordOutcome(pluginID string, sendErr error, isCallerFailure bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.delivered, pluginID)
	switch {
	case sendErr == nil:
		l.delivered[pluginID] = true
		delete(l.rejected, pluginID)
	case !isCallerFailure:
		l.rejected[pluginID] = true
	}
}
