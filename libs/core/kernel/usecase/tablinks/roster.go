package tablinks

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/tablink"
)

// roster is the tabs the windows reported: who owns each, in which order they arrived, and which tabs a
// window saw but another live window owns.
type roster struct {
	tabs        map[string]tablink.Tab
	order       []string
	windows     map[string][]string
	info        map[string]Window
	windowOrder []string
	claims      map[string][]tablink.Tab
	groups      linkGroups
	presence    windowPresence
}

func newRoster() *roster {
	return &roster{
		tabs:     map[string]tablink.Tab{},
		windows:  map[string][]string{},
		info:     map[string]Window{},
		claims:   map[string][]tablink.Tab{},
		groups:   &groupSet{},
		presence: newPresence(),
	}
}

func (r *roster) snapshotTabs() []tablink.Tab {
	out := make([]tablink.Tab, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.tabs[id])
	}
	return out
}

func (r *roster) listWindows(isWatched func(windowID string) bool) []Window {
	out := make([]Window, 0, len(r.windowOrder))
	for _, id := range r.windowOrder {
		if isWatched(id) {
			out = append(out, r.info[id])
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
	if _, ok := r.info[window.ID]; !ok {
		r.windowOrder = append(r.windowOrder, window.ID)
	}
	r.info[window.ID] = window
}

func (r *roster) listClaimers() []string {
	claimers := make([]string, 0, len(r.claims))
	for id := range r.claims {
		claimers = append(claimers, id)
	}
	slices.Sort(claimers)
	return claimers
}

func (r *roster) removeFromWindow(windowID, tabID string) {
	r.windows[windowID] = slices.DeleteFunc(r.windows[windowID], func(id string) bool { return id == tabID })
}

func (r *roster) addToWindow(windowID, tabID string) {
	r.windows[windowID] = appendMissing(r.windows[windowID], tabID)
}

func appendMissing(to []string, ids ...string) []string {
	for _, id := range ids {
		if !slices.Contains(to, id) {
			to = append(to, id)
		}
	}
	return to
}
