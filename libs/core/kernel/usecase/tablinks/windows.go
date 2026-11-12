package tablinks

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// Windows tracks which windows watch the tab links and which tabs each reports.
type Windows struct {
	state shared
}

// Watch subscribes a window to the tab links and cancels the pending removal
// of the window.
func (w *Windows) Watch(windowID string) *Watch {
	s, release := w.state.hold()
	defer release()
	s.expiries.cancelWindowRemoval(s, windowID)
	events, unsubscribe := s.events.Subscribe()
	stop := make(chan struct{})
	watch := &Watch{
		Initial:     s.snapshot(),
		Events:      events,
		Stop:        stop,
		windowID:    windowID,
		stop:        stop,
		unsubscribe: unsubscribe,
		host:        w.state,
	}
	entry := s.roster.ensure(windowID)
	entry.watches = append(entry.watches, watch)
	return watch
}

// Cancel closes Stop of every watch of a window.
func (w *Windows) Cancel(windowID string) {
	s, release := w.state.hold()
	defer release()
	s.roster.windows[windowID].cancelWatches()
}

// GetSnapshot returns every group and every known tab.
func (w *Windows) GetSnapshot() Snapshot {
	s, release := w.state.hold()
	defer release()
	return s.snapshot()
}

// Highlight turns the outline of a window on or off. A window with no watch
// is ignored.
func (w *Windows) Highlight(windowID string, isOn bool) {
	s, release := w.state.hold()
	defer release()
	if !s.roster.windows[windowID].hasWatch() {
		return
	}
	s.events.Publish(Event{Highlight: &Highlight{WindowID: windowID, IsOn: isOn}})
}

// WriteTabs stores the report of a window and its linkable tabs. A report from
// a window with no live watch is ignored. A tab missing from the list is
// unlinked unless it has a pending detach. A tab of another live window moves
// to this window only by Accept; outside a pending move or detach into this
// window it is claimed and moves when the last watch of its owner closes.
func (w *Windows) WriteTabs(window Window, tabs []tablink.Tab) {
	s, release := w.state.hold()
	defer release()
	windowID := window.ID
	if !s.roster.windows[windowID].hasWatch() {
		return
	}
	s.roster.writeWindow(window)
	reported := make([]string, 0, len(tabs))
	var claimed []tablink.Tab
	takenFrom := map[string]string{}
	for _, one := range tabs {
		one.WindowID = windowID
		previous, isKnown := s.roster.tabs[one.ID]
		switch {
		case s.owners.acceptTab(windowID, one):
			reported = append(reported, one.ID)
			if isKnown && previous.WindowID != windowID {
				takenFrom[one.ID] = previous.WindowID
			}
		case !s.transit.isTransferringInto(windowID, one.ID):
			claimed = append(claimed, one)
		}
	}
	for _, id := range s.roster.tabIDsOf(windowID) {
		if !slices.Contains(reported, id) {
			s.forgetTab(id)
		}
	}
	s.roster.setTabIDs(windowID, reported)
	s.roster.ensure(windowID).claims = claimed
	s.owners.dropUnreportedGroupMates(reported, takenFrom)
	s.publishSnapshot()
}
