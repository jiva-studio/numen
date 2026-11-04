// Package tablinks stores the link groups of the tabs of all windows of a vault
// and broadcasts each change to the watching windows.
package tablinks

import (
	"errors"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor/broadcast"
)

// maxColorIndex is the number of group colors; groups past it share colors.
const maxColorIndex = 4

// maxPendingEvents is the queue size of one watch.
const maxPendingEvents = 1024

// Tab is a linkable tab open in a window.
type Tab struct {
	ID, Kind, Title string
	// WindowID is the id of the window that last reported the tab.
	WindowID string
}

// Group is two or more tabs that move to the same path together.
type Group struct {
	ID         string
	ColorIndex int
	TabIDs     []string
	// Path and Title are the last path synced in the group; empty before the
	// first sync. A window that missed a sync moves its tabs to Path from the
	// next snapshot.
	Path, Title string
}

// Snapshot is every group and every known tab.
type Snapshot struct {
	Groups []Group
	Tabs   []Tab
}

// PathSync is the new path of a linked tab.
type PathSync struct {
	GroupID, SourceTabID, Path, Title string
}

// Event is one message to a watching window: a snapshot or a path sync.
type Event struct {
	Snapshot *Snapshot
	Sync     *PathSync
}

// Schedule runs f once after d and returns a function that cancels the run.
type Schedule func(d time.Duration, f func()) (cancel func() bool)

// ScheduleAfter is a Schedule backed by time.AfterFunc.
func ScheduleAfter(d time.Duration, f func()) func() bool { return time.AfterFunc(d, f).Stop }

// Registry stores the link groups of a vault and the linkable tabs of its windows.
type Registry struct {
	grace    time.Duration
	schedule Schedule
	events   broadcast.Broadcaster[Event]

	mu       sync.Mutex
	tabs     map[string]Tab
	tabOrder []string
	// windows maps each window to the tab ids it owns, in report order.
	windows map[string][]string
	// claims maps each window to the tabs it reported while another live
	// window owned them. The window receives such a tab when the owner's last
	// watch closes.
	claims map[string][]Tab
	groups []*Group
	nextID int
	// pendingDetaches is the pending detach of each tab.
	pendingDetaches map[string]*expiry
	// pendingWindowRemovals is the pending removal of each window whose last watch closed.
	pendingWindowRemovals map[string]*expiry
	watches               map[string][]*Watch
}

// NewRegistry returns an empty registry. A detached tab that no window reports
// within grace is unlinked, and so are the tabs of a window that does not watch
// again within grace of its last watch closing.
func NewRegistry(grace time.Duration, schedule Schedule) (*Registry, error) {
	if schedule == nil {
		return nil, errors.New("tab links need a schedule")
	}
	return &Registry{
		grace:    grace,
		schedule: schedule,
		events: broadcast.Broadcaster[Event]{
			IsLatest:  true,
			IsOrdered: true,
			GetTopic:  getEventTopic,
			// A subscriber queues at most one snapshot and one sync per group, so
			// no event is dropped below this many groups.
			Capacity: maxPendingEvents,
		},
		tabs:                  map[string]Tab{},
		windows:               map[string][]string{},
		claims:                map[string][]Tab{},
		pendingDetaches:       map[string]*expiry{},
		pendingWindowRemovals: map[string]*expiry{},
		watches:               map[string][]*Watch{},
	}, nil
}

// getEventTopic returns one topic for all snapshots and one topic per group for
// syncs, so a newer event replaces a pending one on the same topic. The newer
// event is queued last, so a window receives a snapshot and a sync in the order
// they were published.
func getEventTopic(e Event) string {
	if e.Sync != nil {
		return "sync:" + e.Sync.GroupID
	}
	return "snapshot"
}

// Watch is one window's subscription to the registry.
type Watch struct {
	// Initial is the snapshot at watch start; Events delivers every later change.
	Initial Snapshot
	Events  <-chan Event
	// Stop is closed when the window cancels its watch.
	Stop <-chan struct{}

	windowID    string
	stop        chan struct{}
	isCancelled bool
	unsubscribe func()
	registry    *Registry
}

// Close ends the watch. When the window has no other watch, the tabs other
// live windows claimed move to them, and the rest of the window's tabs are
// unlinked unless the window watches again within the grace period.
func (w *Watch) Close() {
	w.unsubscribe()
	r := w.registry
	r.mu.Lock()
	defer r.mu.Unlock()
	rest := slices.DeleteFunc(r.watches[w.windowID], func(one *Watch) bool { return one == w })
	if len(rest) > 0 {
		r.watches[w.windowID] = rest
		return
	}
	delete(r.watches, w.windowID)
	delete(r.claims, w.windowID)
	if r.moveClaimedTabs(w.windowID) {
		r.publishSnapshot()
	}
	pending := &expiry{}
	r.pendingWindowRemovals[w.windowID] = pending
	pending.cancel = r.schedule(r.grace, func() { r.expireWindow(w.windowID, pending) })
}

// Watch subscribes a window to the registry and cancels the pending removal
// of the window.
func (r *Registry) Watch(windowID string) *Watch {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pending, ok := r.pendingWindowRemovals[windowID]; ok {
		pending.cancel()
		delete(r.pendingWindowRemovals, windowID)
	}
	events, unsubscribe := r.events.Subscribe()
	stop := make(chan struct{})
	w := &Watch{
		Initial:     r.getSnapshot(),
		Events:      events,
		Stop:        stop,
		windowID:    windowID,
		stop:        stop,
		unsubscribe: unsubscribe,
		registry:    r,
	}
	r.watches[windowID] = append(r.watches[windowID], w)
	return w
}

// expireWindow unlinks the tabs of a window whose last watch closed. It is a
// no-op when the window watched again.
func (r *Registry) expireWindow(windowID string, pending *expiry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingWindowRemovals[windowID] != pending {
		return
	}
	delete(r.pendingWindowRemovals, windowID)
	r.removeWindow(windowID)
}

// moveClaimedTabs gives each tab of a window to a live window that claimed it
// and returns whether a tab moved.
func (r *Registry) moveClaimedTabs(ownerID string) bool {
	isMoved := false
	claimers := make([]string, 0, len(r.claims))
	for id := range r.claims {
		claimers = append(claimers, id)
	}
	slices.Sort(claimers)
	for _, claimerID := range claimers {
		r.claims[claimerID] = slices.DeleteFunc(r.claims[claimerID], func(tab Tab) bool {
			if r.tabs[tab.ID].WindowID != ownerID {
				return false
			}
			r.setOwner(claimerID, tab)
			r.windows[claimerID] = append(r.windows[claimerID], tab.ID)
			isMoved = true
			return true
		})
	}
	return isMoved
}

// Cancel closes Stop of every watch of a window.
func (r *Registry) Cancel(windowID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, w := range r.watches[windowID] {
		if !w.isCancelled {
			w.isCancelled = true
			close(w.stop)
		}
	}
}

// GetSnapshot returns every group and every known tab.
func (r *Registry) GetSnapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getSnapshot()
}

// WriteTabs sets the linkable tabs of a window. A window with no live watch is
// ignored. A tab of the window missing from the list is unlinked unless it has
// a pending detach. A tab of another live window is reassigned to this window
// during a detach, and outside a detach is claimed: it moves to this window when the
// last watch of its owner closes.
func (r *Registry) WriteTabs(windowID string, tabs []Tab) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.watches[windowID]) == 0 {
		return
	}
	reported := make([]string, 0, len(tabs))
	var claimed []Tab
	for _, one := range tabs {
		one.WindowID = windowID
		if r.acceptTab(windowID, one) {
			reported = append(reported, one.ID)
		} else {
			claimed = append(claimed, one)
		}
	}
	for _, id := range r.windows[windowID] {
		if !slices.Contains(reported, id) {
			r.forgetTab(id)
		}
	}
	r.windows[windowID] = reported
	r.claims[windowID] = claimed
	r.publishSnapshot()
}

// acceptTab records a tab reported by a window and returns whether the tab now
// belongs to that window.
func (r *Registry) acceptTab(windowID string, tab Tab) bool {
	previous, isKnown := r.tabs[tab.ID]
	switch {
	case !isKnown:
		r.tabOrder = append(r.tabOrder, tab.ID)
	case previous.WindowID == windowID:
	case r.pendingDetaches[tab.ID] == nil && len(r.watches[previous.WindowID]) > 0:
		return false
	}
	r.setOwner(windowID, tab)
	return true
}

// setOwner records a tab as a tab of a window, ends its pending detach, and
// takes it out of the list of its previous window.
func (r *Registry) setOwner(windowID string, tab Tab) {
	tab.WindowID = windowID
	previous, isKnown := r.tabs[tab.ID]
	r.tabs[tab.ID] = tab
	if !isKnown || previous.WindowID == windowID {
		return
	}
	if pending, ok := r.pendingDetaches[tab.ID]; ok {
		pending.cancel()
		delete(r.pendingDetaches, tab.ID)
	}
	r.windows[previous.WindowID] = slices.DeleteFunc(r.windows[previous.WindowID], func(id string) bool { return id == tab.ID })
}

// Link puts two known tabs in one group. If a tab is already in a group, the
// other tab joins it; two groups merge.
func (r *Registry) Link(sourceID, targetID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, isSourceKnown := r.tabs[sourceID]
	_, isTargetKnown := r.tabs[targetID]
	if sourceID == targetID || !isSourceKnown || !isTargetKnown {
		return
	}
	source, target := r.findGroup(sourceID), r.findGroup(targetID)
	switch {
	case source != nil && source == target:
		return
	case source != nil && target != nil:
		source.TabIDs = appendMissing(source.TabIDs, target.TabIDs...)
		if source.Path == "" {
			source.Path, source.Title = target.Path, target.Title
		}
		r.groups = slices.DeleteFunc(r.groups, func(g *Group) bool { return g == target })
	case source != nil:
		source.TabIDs = append(source.TabIDs, targetID)
	case target != nil:
		target.TabIDs = append(target.TabIDs, sourceID)
	default:
		r.nextID++
		r.groups = append(r.groups, &Group{
			ID:         "group-" + strconv.Itoa(r.nextID),
			ColorIndex: r.getNextColorIndex(),
			TabIDs:     []string{sourceID, targetID},
		})
	}
	r.publishSnapshot()
}

// Unlink removes a tab from its group.
func (r *Registry) Unlink(tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.removeFromGroup(tabID) {
		r.publishSnapshot()
	}
}

// Detach keeps a tab in its group while it moves to a new window. A tab that
// no window reports within the grace period is unlinked.
func (r *Registry) Detach(tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tabs[tabID]; !ok {
		return
	}
	if previous, ok := r.pendingDetaches[tabID]; ok {
		previous.cancel()
	}
	pending := &expiry{}
	r.pendingDetaches[tabID] = pending
	pending.cancel = r.schedule(r.grace, func() { r.expireDetach(tabID, pending) })
}

// expiry is a pending detach of a tab or a pending removal of a window, and the
// cancel func of its run.
type expiry struct{ cancel func() bool }

// CancelDetach cancels the pending detach of a tab. A tab missing from the last
// report of its window is unlinked.
func (r *Registry) CancelDetach(tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pending, ok := r.pendingDetaches[tabID]
	if !ok {
		return
	}
	pending.cancel()
	delete(r.pendingDetaches, tabID)
	r.dropUnreportedTab(tabID)
}

// expireDetach removes a detached tab that no window reported. It is a no-op
// when a report or a newer Detach replaced this detach.
func (r *Registry) expireDetach(tabID string, pending *expiry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingDetaches[tabID] != pending {
		return
	}
	delete(r.pendingDetaches, tabID)
	r.dropUnreportedTab(tabID)
}

// dropUnreportedTab removes a tab missing from the last report of its window.
func (r *Registry) dropUnreportedTab(tabID string) {
	if slices.Contains(r.windows[r.tabs[tabID].WindowID], tabID) {
		return
	}
	r.dropTab(tabID)
	r.publishSnapshot()
}

// SyncPath records the new path of a linked tab as the path of its group and
// sends it to every watching window. It returns whether the tab is in a group.
func (r *Registry) SyncPath(sourceTabID, path, title string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	group := r.findGroup(sourceTabID)
	if group == nil {
		return false
	}
	group.Path, group.Title = path, title
	r.events.Publish(Event{Sync: &PathSync{GroupID: group.ID, SourceTabID: sourceTabID, Path: path, Title: title}})
	return true
}

// removeWindow unlinks the tabs of a window, except tabs with a pending
// detach.
func (r *Registry) removeWindow(windowID string) {
	ids, ok := r.windows[windowID]
	if !ok {
		return
	}
	delete(r.windows, windowID)
	for _, id := range ids {
		r.forgetTab(id)
	}
	r.publishSnapshot()
}

// forgetTab removes a tab missing from the report of its window, unless it has a
// pending detach.
func (r *Registry) forgetTab(tabID string) {
	if _, ok := r.pendingDetaches[tabID]; ok {
		return
	}
	r.dropTab(tabID)
}

func (r *Registry) dropTab(tabID string) {
	delete(r.tabs, tabID)
	r.tabOrder = slices.DeleteFunc(r.tabOrder, func(id string) bool { return id == tabID })
	r.removeFromGroup(tabID)
}

// removeFromGroup removes a tab from its group and deletes a group left with
// fewer than two tabs. It returns whether the tab was in a group.
func (r *Registry) removeFromGroup(tabID string) bool {
	group := r.findGroup(tabID)
	if group == nil {
		return false
	}
	group.TabIDs = slices.DeleteFunc(group.TabIDs, func(id string) bool { return id == tabID })
	if len(group.TabIDs) < 2 {
		r.groups = slices.DeleteFunc(r.groups, func(g *Group) bool { return g == group })
	}
	return true
}

func (r *Registry) findGroup(tabID string) *Group {
	for _, group := range r.groups {
		if slices.Contains(group.TabIDs, tabID) {
			return group
		}
	}
	return nil
}

// getNextColorIndex returns the lowest color no group has, or 1 when every
// color is taken.
func (r *Registry) getNextColorIndex() int {
	for index := 1; index <= maxColorIndex; index++ {
		if !slices.ContainsFunc(r.groups, func(g *Group) bool { return g.ColorIndex == index }) {
			return index
		}
	}
	return 1
}

func (r *Registry) getSnapshot() Snapshot {
	out := Snapshot{Groups: make([]Group, 0, len(r.groups)), Tabs: make([]Tab, 0, len(r.tabOrder))}
	for _, group := range r.groups {
		copied := *group
		copied.TabIDs = slices.Clone(group.TabIDs)
		out.Groups = append(out.Groups, copied)
	}
	for _, id := range r.tabOrder {
		out.Tabs = append(out.Tabs, r.tabs[id])
	}
	return out
}

func (r *Registry) publishSnapshot() {
	snapshot := r.getSnapshot()
	r.events.Publish(Event{Snapshot: &snapshot})
}

func appendMissing(to []string, ids ...string) []string {
	for _, id := range ids {
		if !slices.Contains(to, id) {
			to = append(to, id)
		}
	}
	return to
}
