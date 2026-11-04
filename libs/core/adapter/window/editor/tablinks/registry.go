// Package tablinks stores the link groups of the tabs of all windows of a vault
// and broadcasts each change to the watching windows.
package tablinks

import (
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
	// windows maps each window to the tab ids it last reported, in tab order.
	windows map[string][]string
	groups  []*Group
	nextID  int
	// detaching is the set of tabs with a pending detach.
	detaching map[string]*detach
	watches   map[string][]*Watch
}

// NewRegistry returns an empty registry. A detached tab that no window reports
// within grace is unlinked.
func NewRegistry(grace time.Duration, schedule Schedule) *Registry {
	return &Registry{
		grace:    grace,
		schedule: schedule,
		events: broadcast.Broadcaster[Event]{
			IsLatest: true,
			GetTopic: getEventTopic,
			// A subscriber queues at most one snapshot and one sync per group, so
			// no event is dropped below this many groups.
			Capacity: maxPendingEvents,
		},
		tabs:      map[string]Tab{},
		windows:   map[string][]string{},
		detaching: map[string]*detach{},
		watches:   map[string][]*Watch{},
	}
}

// getEventTopic returns one topic for all snapshots and one topic per group for
// syncs, so a newer event replaces a pending one on the same topic.
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

// Close ends the watch. When the window has no other watch, its tabs are unlinked.
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
	r.removeWindow(w.windowID)
}

// Watch subscribes a window to the registry.
func (r *Registry) Watch(windowID string) *Watch {
	r.mu.Lock()
	defer r.mu.Unlock()
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
// only during a detach.
func (r *Registry) WriteTabs(windowID string, tabs []Tab) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.watches[windowID]) == 0 {
		return
	}
	reported := make([]string, 0, len(tabs))
	for _, one := range tabs {
		if r.acceptTab(windowID, one) {
			reported = append(reported, one.ID)
		}
	}
	for _, id := range r.windows[windowID] {
		if !slices.Contains(reported, id) {
			r.forgetTab(id)
		}
	}
	r.windows[windowID] = reported
	r.publishSnapshot()
}

// acceptTab records a tab reported by a window and returns whether the tab now
// belongs to that window.
func (r *Registry) acceptTab(windowID string, tab Tab) bool {
	tab.WindowID = windowID
	previous, isKnown := r.tabs[tab.ID]
	pending, isDetaching := r.detaching[tab.ID]
	switch {
	case !isKnown:
		r.tabOrder = append(r.tabOrder, tab.ID)
	case previous.WindowID == windowID:
	case isDetaching:
		pending.cancel()
		delete(r.detaching, tab.ID)
		r.windows[previous.WindowID] = slices.DeleteFunc(r.windows[previous.WindowID], func(id string) bool { return id == tab.ID })
	case len(r.watches[previous.WindowID]) > 0:
		return false
	}
	r.tabs[tab.ID] = tab
	return true
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
	if previous, ok := r.detaching[tabID]; ok {
		previous.cancel()
	}
	pending := &detach{}
	r.detaching[tabID] = pending
	pending.cancel = r.schedule(r.grace, func() { r.expireDetach(tabID, pending) })
}

// detach is a pending detach of a tab and the cancel func of its expiry.
type detach struct{ cancel func() bool }

// expireDetach removes a detached tab that no window reported. It is a no-op
// when a report or a newer Detach replaced this detach.
func (r *Registry) expireDetach(tabID string, pending *detach) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.detaching[tabID] != pending {
		return
	}
	delete(r.detaching, tabID)
	tab := r.tabs[tabID]
	if slices.Contains(r.windows[tab.WindowID], tabID) {
		return
	}
	r.dropTab(tabID)
	r.publishSnapshot()
}

// SyncPath sends the new path of a linked tab to every watching window. It
// returns whether the tab is in a group.
func (r *Registry) SyncPath(sourceTabID, path, title string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	group := r.findGroup(sourceTabID)
	if group == nil {
		return false
	}
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
	if _, ok := r.detaching[tabID]; ok {
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
		out.Groups = append(out.Groups, Group{ID: group.ID, ColorIndex: group.ColorIndex, TabIDs: slices.Clone(group.TabIDs)})
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
