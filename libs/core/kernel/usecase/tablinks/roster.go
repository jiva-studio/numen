package tablinks

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// roster is the tabs, the windows that report them, and the link groups of the tabs. windowOrder is the order
// in which the windows were first listed.
type roster struct {
	tabs        map[string]tablink.Tab
	order       []string
	windows     map[string]*windowEntry
	windowOrder []string
	groups      linkGroups
}

// windowEntry is what the state knows of one window. isReported is whether the window ever had its tabs
// set, and isListed whether it ever had its info written.
type windowEntry struct {
	tabIDs     []string
	isReported bool
	info       Window
	isListed   bool
	claims     []tablink.Tab
	watches    []*Watch
	removal    *expiry
}

func newRoster() *roster {
	return &roster{
		tabs:    map[string]tablink.Tab{},
		windows: map[string]*windowEntry{},
		groups:  &groupSet{},
	}
}

// ensure returns the entry of the window, which it makes where there is none.
func (r *roster) ensure(windowID string) *windowEntry {
	entry := r.windows[windowID]
	if entry == nil {
		entry = &windowEntry{}
		r.windows[windowID] = entry
	}
	return entry
}

// prune forgets the entry of the window when it holds nothing.
func (r *roster) prune(windowID string) {
	if entry := r.windows[windowID]; entry != nil && entry.isEmpty() {
		delete(r.windows, windowID)
	}
}

func (r *roster) snapshotTabs() []tablink.Tab {
	out := make([]tablink.Tab, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.tabs[id])
	}
	return out
}

func (r *roster) listWindows() []Window {
	out := make([]Window, 0, len(r.windowOrder))
	for _, id := range r.windowOrder {
		if entry := r.windows[id]; entry.hasWatch() {
			out = append(out, entry.info)
		}
	}
	slices.SortStableFunc(out, func(a, b Window) int {
		switch {
		case a.IsMain == b.IsMain:
			return 0
		case a.IsMain:
			return -1
		default:
			return 1
		}
	})
	return out
}

func (r *roster) writeWindow(window Window) {
	entry := r.ensure(window.ID)
	if !entry.isListed {
		r.windowOrder = append(r.windowOrder, window.ID)
		entry.isListed = true
	}
	entry.info = window
}

func (r *roster) listClaimers() []string {
	var claimers []string
	for id, entry := range r.windows {
		if len(entry.claims) > 0 {
			claimers = append(claimers, id)
		}
	}
	slices.Sort(claimers)
	return claimers
}

func (r *roster) tabIDsOf(windowID string) []string {
	if entry := r.windows[windowID]; entry != nil {
		return entry.tabIDs
	}
	return nil
}

func (r *roster) setTabIDs(windowID string, ids []string) {
	entry := r.ensure(windowID)
	entry.tabIDs, entry.isReported = ids, true
}

func (r *roster) removeFromWindow(windowID, tabID string) {
	r.setTabIDs(windowID, slices.DeleteFunc(r.tabIDsOf(windowID), func(id string) bool { return id == tabID }))
}

func (r *roster) addToWindow(windowID, tabID string) {
	r.setTabIDs(windowID, appendMissing(r.tabIDsOf(windowID), tabID))
}

func appendMissing(to []string, ids ...string) []string {
	for _, id := range ids {
		if !slices.Contains(to, id) {
			to = append(to, id)
		}
	}
	return to
}
