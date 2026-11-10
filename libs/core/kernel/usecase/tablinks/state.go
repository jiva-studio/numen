package tablinks

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
	"github.com/jiva-studio/numen/libs/core/kernel/port/tablinkport"
)

// Tabs are the use cases of the tab links of one vault.
type Tabs struct {
	Links     *Links
	Windows   *Windows
	Transfers *Transfers
}

// New returns the use cases over an empty state. A detached tab that no window reports within grace is
// unlinked, and so are the tabs of a window that does not watch again within grace of its last watch
// closing.
func New(grace time.Duration, schedule tablinkport.Schedule, events tablinkport.Events) (*Tabs, error) {
	if schedule == nil {
		return nil, errors.New("tab links need a schedule")
	}
	s := &state{
		events:   events,
		roster:   newRoster(),
		transit:  newTransit(),
		presence: newPresence(),
	}
	s.owners = &ownership{state: s}
	s.expiries = &expiries{state: s, grace: grace, schedule: schedule}
	return &Tabs{Links: &Links{state: s}, Windows: &Windows{state: s}, Transfers: &Transfers{state: s}}, nil
}

// state is what the use cases share; mu guards every field of it and of its parts.
type state struct {
	mu       sync.Mutex
	events   tablinkport.Events
	roster   *roster
	transit  *transit
	presence *presence
	owners   *ownership
	expiries *expiries
}

func (s *state) snapshot() tablink.Snapshot {
	return tablink.Snapshot{
		Groups:  s.roster.groups.snapshot(),
		Tabs:    s.roster.snapshotTabs(),
		Windows: s.roster.listWindows(s.presence.hasWatch),
	}
}

func (s *state) publishSnapshot() {
	snapshot := s.snapshot()
	s.events.Publish(tablink.Event{Snapshot: &snapshot})
}

func (s *state) removeWindow(windowID string) {
	ids, ok := s.roster.windows[windowID]
	if !ok {
		return
	}
	delete(s.roster.windows, windowID)
	delete(s.roster.info, windowID)
	s.roster.windowOrder = slices.DeleteFunc(s.roster.windowOrder, func(id string) bool { return id == windowID })
	for _, id := range ids {
		s.forgetTab(id)
	}
	s.publishSnapshot()
}

func (s *state) forgetTab(tabID string) {
	if _, ok := s.transit.detaches[tabID]; ok {
		return
	}
	s.dropTab(tabID)
}

func (s *state) dropTab(tabID string) {
	delete(s.roster.tabs, tabID)
	s.roster.order = slices.DeleteFunc(s.roster.order, func(id string) bool { return id == tabID })
	s.roster.groups.remove(tabID)
}

func (s *state) dropUnreportedTab(tabID string) {
	tab, isKnown := s.roster.tabs[tabID]
	if !isKnown || slices.Contains(s.roster.windows[tab.WindowID], tabID) {
		return
	}
	s.dropTab(tabID)
	s.publishSnapshot()
}
