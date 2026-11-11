package tablinks

import "slices"

// expiry is a pending removal of a window and the cancel func of its run.
// hasAccepted marks a window that accepted a tab while it had no watch.
type expiry struct {
	cancel      func() bool
	hasAccepted bool
}

func (e *windowEntry) hasWatch() bool { return e != nil && len(e.watches) > 0 }

func (e *windowEntry) isLive() bool {
	return e.hasWatch() || (e != nil && e.removal != nil && e.removal.hasAccepted)
}

func (e *windowEntry) isEmpty() bool {
	return !e.isReported && !e.isListed && len(e.claims) == 0 && len(e.watches) == 0 && e.removal == nil
}

func (e *windowEntry) removeWatch(w *Watch) bool {
	if e == nil {
		return true
	}
	e.watches = slices.DeleteFunc(e.watches, func(one *Watch) bool { return one == w })
	return len(e.watches) == 0
}

func (e *windowEntry) cancelWatches() {
	if e == nil {
		return
	}
	for _, w := range e.watches {
		if !w.isCanceled {
			w.isCanceled = true
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
	isCanceled  bool
	unsubscribe func()
	host        shared
}

// Close ends the watch. When the window has no other watch, the pending moves
// into it end with MoveErrorWindowClosed, the window leaves the window list,
// the tabs other live windows claimed move to them, and the rest of the
// window's tabs are unlinked unless the window watches again within the grace
// period.
func (w *Watch) Close() {
	w.unsubscribe()
	s, release := w.host.hold()
	defer release()
	if !s.roster.windows[w.windowID].removeWatch(w) {
		return
	}
	s.transit.endMovesInto(w.windowID)
	if entry := s.roster.windows[w.windowID]; entry != nil {
		entry.claims = nil
	}
	s.owners.moveClaimedTabs(w.windowID)
	s.publishSnapshot()
	s.expiries.scheduleWindowRemoval(s, w.windowID)
}
