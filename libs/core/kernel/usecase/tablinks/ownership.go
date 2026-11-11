package tablinks

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// ownership changes which window owns a tab.
type ownership struct {
	state *state
}

func (o *ownership) setOwner(windowID string, tab tablink.Tab) {
	s := o.state
	tab.WindowID = windowID
	previous, isKnown := s.roster.tabs[tab.ID]
	s.roster.tabs[tab.ID] = tab
	if !isKnown || previous.WindowID == windowID {
		return
	}
	if pending, ok := s.transit.getMove(tab.ID); ok && pending.targetWindowID != windowID {
		s.transit.endMove(tab.ID, pending, tablink.MoveErrorCancelled)
	}
	if pending, ok := s.transit.getDetach(tab.ID); ok {
		outcome := tablink.MoveErrorCancelled
		if pending.windowID == windowID {
			// The window that read the detach still calls Accept.
			outcome = tablink.MoveErrorNone
			s.expiries.recordAccepted(pending.id, windowID)
		}
		s.transit.endDetach(tab.ID, pending, outcome)
	}
	s.roster.removeFromWindow(previous.WindowID, tab.ID)
}

func (o *ownership) acceptTab(windowID string, tab tablink.Tab) bool {
	s := o.state
	previous, isKnown := s.roster.tabs[tab.ID]
	switch {
	case !isKnown:
		s.roster.order = append(s.roster.order, tab.ID)
	case previous.WindowID == windowID:
	case s.roster.presence.isLive(previous.WindowID):
		return false
	}
	o.setOwner(windowID, tab)
	return true
}

func (o *ownership) takeTab(windowID, tabID string) {
	s := o.state
	tab, isKnown := s.roster.tabs[tabID]
	if !isKnown {
		return
	}
	s.expiries.keepAcceptingWindow(windowID)
	o.setOwner(windowID, tab)
	s.roster.addToWindow(windowID, tabID)
	s.publishSnapshot()
}

func (o *ownership) moveClaimedTabs(ownerID string) {
	s := o.state
	for _, claimerID := range s.roster.listClaimers() {
		s.roster.claims[claimerID] = slices.DeleteFunc(s.roster.claims[claimerID], func(tab tablink.Tab) bool {
			if s.roster.tabs[tab.ID].WindowID != ownerID {
				return false
			}
			o.setOwner(claimerID, tab)
			s.roster.windows[claimerID] = append(s.roster.windows[claimerID], tab.ID)
			return true
		})
	}
}

func (o *ownership) dropUnreportedGroupMates(reported []string, takenFrom map[string]string) {
	s := o.state
	for tabID, oldWindowID := range takenFrom {
		group := s.roster.groups.find(tabID)
		if group == nil {
			continue
		}
		for _, mate := range slices.Clone(group.TabIDs) {
			if slices.Contains(reported, mate) || s.roster.tabs[mate].WindowID != oldWindowID {
				continue
			}
			s.forgetTab(mate)
		}
	}
}
