package delivery

import "sync"

// ledger remembers, for each plugin, which list of secrets it was given, which list it refused, and
// which plugin has not been configured since it started.
type ledger struct {
	mu                     sync.RWMutex
	delivered              map[string]string
	rejected               map[string]string
	awaitingFirstConfigure map[string]bool
}

func newLedger() *ledger {
	return &ledger{
		delivered:              make(map[string]string),
		rejected:               make(map[string]string),
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

func (l *ledger) isDue(pluginID, signature string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.awaitingFirstConfigure[pluginID] {
		return false
	}
	if last, isDelivered := l.delivered[pluginID]; isDelivered && last == signature {
		return false
	}
	last, isRejected := l.rejected[pluginID]
	return !isRejected || last != signature
}

func (l *ledger) recordOutcome(pluginID, signature string, sendErr, readErr error, isCallerFailure bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch {
	case sendErr == nil && readErr == nil:
		l.delivered[pluginID] = signature
		delete(l.rejected, pluginID)
	case sendErr == nil:
		delete(l.delivered, pluginID)
		delete(l.rejected, pluginID)
	case isCallerFailure:
		delete(l.delivered, pluginID)
	default:
		delete(l.delivered, pluginID)
		l.rejected[pluginID] = signature
	}
}
