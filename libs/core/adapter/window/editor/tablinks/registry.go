// Package tablinks keeps the link groups of the tabs of every window of a
// vault, and fans each change out to the windows that watch them.
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

// maxPendingEvents bounds the events waiting for one watch.
const maxPendingEvents = 1024

// Tab is a linkable tab a window has open.
type Tab struct {
	ID, Kind, Title string
	// WindowID is the window that last reported the tab.
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

// PathSync is a linked tab that moved to a path.
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

// ScheduleAfter is the Schedule of the machine's timers.
func ScheduleAfter(d time.Duration, f func()) func() bool { return time.AfterFunc(d, f).Stop }

// Registry is the link groups of a vault and the linkable tabs of its windows.
type Registry struct {
	grace    time.Duration
	schedule Schedule
	events   broadcast.Broadcaster[Event]

	mu       sync.Mutex
	tabs     map[string]Tab
	tabOrder []string
	// windows is the tab ids each window last reported, in its order.
	windows map[string][]string
	groups  []*Group
	nextID  int
	// detaching is the tabs a window was asked to detach.
	detaching map[string]*detach
	watches   map[string][]*Watch
}

// NewRegistry returns an empty registry. A detached tab that no window reports
// within grace leaves its group.
func NewRegistry(grace time.Duration, schedule Schedule) *Registry {
	return &Registry{
		grace:    grace,
		schedule: schedule,
		events: broadcast.Broadcaster[Event]{
			IsLatest: true,
			GetTopic: getEventTopic,
			// A subscriber holds at most one snapshot and one sync per group, so
			// nothing is dropped below this many groups.
			Capacity: maxPendingEvents,
		},
		tabs:      map[string]Tab{},
		windows:   map[string][]string{},
		detaching: map[string]*detach{},
		watches:   map[string][]*Watch{},
	}
}

// getEventTopic puts every snapshot on one topic, so a pending snapshot is
// replaced by a newer one, and each group's syncs on a topic of their own.
func getEventTopic(e Event) string {
	if e.Sync != nil {
		return "sync:" + e.Sync.GroupID
	}
	return "snapshot"
}

// Watch is one window's subscription to the registry.
type Watch struct {
	// Initial is the state when the watch began; Events carries every change after it.
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

// Close ends the watch. The tabs of a window with no watch left leave their groups.
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

// WriteTabs sets the linkable tabs a window has open. A window with no live
// watch is ignored. A tab of the window missing from the list leaves its group
// unless it is being detached. A tab another live window has moves to this
// window only while it is being detached.
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

// acceptTab records a tab a window reports, and reports whether the tab is now
// the window's.
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

// Link puts two known tabs in one group. A tab already in a group brings its
// group along; two groups merge.
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

// Unlink takes a tab out of its group.
func (r *Registry) Unlink(tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.removeFromGroup(tabID) {
		r.publishSnapshot()
	}
}

// Detach keeps a tab in its group while it moves to a new window. A tab that
// no window reports within the grace period leaves its group.
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

// detach is one detach of a tab and the cancel of its expiry.
type detach struct{ cancel func() bool }

// expireDetach drops a detached tab that no window reports. An expiry that a
// report or a newer Detach replaced does nothing.
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

// SyncPath sends a path a linked tab moved to to every watching window. It
// reports whether the tab is in a group.
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

// removeWindow drops the tabs of a window from their groups, except the tabs
// being detached.
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

// forgetTab drops a tab its window no longer has, unless it is being detached.
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

// removeFromGroup takes a tab out of its group, and drops a group left with
// fewer than two tabs. It reports whether the tab was in a group.
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
