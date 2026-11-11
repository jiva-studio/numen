package tablinks

import "slices"

// expiry is a pending removal of a window and the cancel func of its run.
// hasAccepted marks a window that accepted a tab while it had no watch.
type expiry struct {
	cancel      func() bool
	hasAccepted bool
}

// presence is which windows are watching, and the windows whose last watch ended and wait out the grace
// period.
type presence struct {
	watches  map[string][]*Watch
	removals map[string]*expiry
}

func newPresence() *presence {
	return &presence{watches: map[string][]*Watch{}, removals: map[string]*expiry{}}
}

func (p *presence) hasWatch(windowID string) bool { return len(p.watches[windowID]) > 0 }

func (p *presence) isLive(windowID string) bool {
	if p.hasWatch(windowID) {
		return true
	}
	pending, ok := p.removals[windowID]
	return ok && pending.hasAccepted
}

func (p *presence) addWatch(w *Watch) {
	p.watches[w.windowID] = append(p.watches[w.windowID], w)
}

func (p *presence) removeWatch(w *Watch) bool {
	rest := slices.DeleteFunc(p.watches[w.windowID], func(one *Watch) bool { return one == w })
	if len(rest) > 0 {
		p.watches[w.windowID] = rest
		return false
	}
	delete(p.watches, w.windowID)
	return true
}

func (p *presence) cancelWatches(windowID string) {
	for _, w := range p.watches[windowID] {
		if !w.isCancelled {
			w.isCancelled = true
			close(w.stop)
		}
	}
}

// Watch is one window's subscription to the tab links.
type Watch struct {
	Initial Snapshot
	Events  <-chan Event
	Stop    <-chan struct{}

	windowID    string
	stop        chan struct{}
	isCancelled bool
	unsubscribe func()
	windows     *Windows
}

// Close ends the watch. When the window has no other watch, the pending moves
// into it end with MoveErrorWindowClosed, the window leaves the window list,
// the tabs other live windows claimed move to them, and the rest of the
// window's tabs are unlinked unless the window watches again within the grace
// period.
func (w *Watch) Close() {
	w.unsubscribe()
	s := w.windows.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.roster.presence.removeWatch(w) {
		return
	}
	s.transit.endMovesInto(w.windowID)
	delete(s.roster.claims, w.windowID)
	s.owners.moveClaimedTabs(w.windowID)
	s.publishSnapshot()
	s.expiries.scheduleWindowRemoval(w.windowID)
}
