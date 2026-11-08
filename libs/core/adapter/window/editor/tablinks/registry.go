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
	WindowID        string
}

// Group is two or more tabs that move to the same path together.
type Group struct {
	ID          string
	ColorIndex  int
	TabIDs      []string
	Path, Title string
}

// Window is the last report of a window about itself.
type Window struct {
	ID                      string
	IsMain                  bool
	ActiveTitle, ActiveKind string
	TabCount                int
}

// Snapshot holds every group, every known tab and every window that reported.
type Snapshot struct {
	Groups  []Group
	Tabs    []Tab
	Windows []Window
}

// PathSync is the new path of a linked tab.
type PathSync struct {
	GroupID, SourceTabID, Path, Title string
}

// Event is one message to watching windows: a snapshot or a path sync for
// every window, or a moved tab or a highlight for one.
type Event struct {
	Snapshot  *Snapshot
	Sync      *PathSync
	Move      *MovedTab
	Highlight *Highlight
}

// IsFor reports whether a window receives the event.
func (e Event) IsFor(windowID string) bool {
	switch {
	case e.Move != nil:
		return e.Move.WindowID == windowID
	case e.Highlight != nil:
		return e.Highlight.WindowID == windowID
	default:
		return true
	}
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

	mu                    sync.Mutex
	tabs                  map[string]Tab
	tabOrder              []string
	windows               map[string][]string
	windowInfo            map[string]Window
	windowOrder           []string
	claims                map[string][]Tab
	groups                []*Group
	nextID                int
	pendingDetaches       map[string]*detach
	pendingWindowRemovals map[string]*expiry
	pendingMoves          map[string]*move
	nextMoveID            int
	acceptedMoves         map[string]string
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
		windowInfo:            map[string]Window{},
		claims:                map[string][]Tab{},
		pendingDetaches:       map[string]*detach{},
		pendingWindowRemovals: map[string]*expiry{},
		pendingMoves:          map[string]*move{},
		acceptedMoves:         map[string]string{},
		watches:               map[string][]*Watch{},
	}, nil
}

func getEventTopic(e Event) string {
	switch {
	case e.Sync != nil:
		return "sync:" + e.Sync.GroupID
	case e.Move != nil:
		return "move:" + e.Move.Tab.ID
	case e.Highlight != nil:
		return "highlight:" + e.Highlight.WindowID
	default:
		return "snapshot"
	}
}

// Watch is one window's subscription to the registry.
type Watch struct {
	Initial Snapshot
	Events  <-chan Event
	Stop    <-chan struct{}

	windowID    string
	stop        chan struct{}
	isCancelled bool
	unsubscribe func()
	registry    *Registry
}

// Close ends the watch. When the window has no other watch, the pending moves
// into it end with MoveErrorWindowClosed, the window leaves the window list,
// the tabs other live windows claimed move to them, and the rest of the
// window's tabs are unlinked unless the window watches again within the grace
// period.
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
	r.endMovesInto(w.windowID)
	delete(r.claims, w.windowID)
	r.moveClaimedTabs(w.windowID)
	r.publishSnapshot()
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

func (r *Registry) expireWindow(windowID string, pending *expiry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingWindowRemovals[windowID] != pending {
		return
	}
	delete(r.pendingWindowRemovals, windowID)
	r.removeWindow(windowID)
}

func (r *Registry) moveClaimedTabs(ownerID string) {
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
			return true
		})
	}
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

// WriteTabs stores the report of a window and its linkable tabs. A report from
// a window with no live watch is ignored. A tab of the window missing from the
// list is unlinked unless it has a pending detach. A tab of another live window
// moves to this window only by Accept. Outside a pending move or detach into
// this window, it is claimed: it moves to this window when the last watch of
// its owner closes.
func (r *Registry) WriteTabs(window Window, tabs []Tab) {
	r.mu.Lock()
	defer r.mu.Unlock()
	windowID := window.ID
	if len(r.watches[windowID]) == 0 {
		return
	}
	if _, ok := r.windowInfo[windowID]; !ok {
		r.windowOrder = append(r.windowOrder, windowID)
	}
	r.windowInfo[windowID] = window
	reported := make([]string, 0, len(tabs))
	var claimed []Tab
	takenFrom := map[string]string{}
	for _, one := range tabs {
		one.WindowID = windowID
		previous, isKnown := r.tabs[one.ID]
		switch {
		case r.acceptTab(windowID, one):
			reported = append(reported, one.ID)
			if isKnown && previous.WindowID != windowID {
				takenFrom[one.ID] = previous.WindowID
			}
		case !r.isMovingInto(windowID, one.ID) && !r.isDetachingInto(windowID, one.ID):
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
	r.dropUnreportedGroupMates(reported, takenFrom)
	r.publishSnapshot()
}

func (r *Registry) dropUnreportedGroupMates(reported []string, takenFrom map[string]string) {
	for tabID, oldWindowID := range takenFrom {
		group := r.findGroup(tabID)
		if group == nil {
			continue
		}
		for _, mate := range slices.Clone(group.TabIDs) {
			if slices.Contains(reported, mate) || r.tabs[mate].WindowID != oldWindowID {
				continue
			}
			r.forgetTab(mate)
		}
	}
}

func (r *Registry) acceptTab(windowID string, tab Tab) bool {
	previous, isKnown := r.tabs[tab.ID]
	switch {
	case !isKnown:
		r.tabOrder = append(r.tabOrder, tab.ID)
	case previous.WindowID == windowID:
	case r.isLive(previous.WindowID):
		return false
	}
	r.setOwner(windowID, tab)
	return true
}

func (r *Registry) setOwner(windowID string, tab Tab) {
	tab.WindowID = windowID
	previous, isKnown := r.tabs[tab.ID]
	r.tabs[tab.ID] = tab
	if !isKnown || previous.WindowID == windowID {
		return
	}
	if pending, ok := r.pendingMoves[tab.ID]; ok && pending.targetWindowID != windowID {
		r.endMove(tab.ID, pending, MoveErrorCancelled)
	}
	if pending, ok := r.pendingDetaches[tab.ID]; ok {
		outcome := MoveErrorCancelled
		if pending.windowID == windowID {
			// The window that read the detach still calls Accept.
			outcome = MoveErrorNone
			r.recordAcceptedMove(pending.id, windowID)
		}
		r.endDetach(tab.ID, pending, outcome)
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

// Detach stores a tab for ReadDetached and keeps the tab in its group while it
// moves to a new window. It returns a channel that receives the result once:
// MoveErrorNone when the window that read the tab calls Accept. A tab that no
// window reports within the grace period is dropped and unlinked. It ends a
// pending move or detach of the same tab with MoveErrorCancelled.
func (r *Registry) Detach(tab SavedTab) <-chan MoveError {
	result := make(chan MoveError, 1)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endTransfers(tab.ID, MoveErrorCancelled)
	pending := &detach{id: r.createMoveID(), tab: tab, result: result}
	r.pendingDetaches[tab.ID] = pending
	pending.cancel = r.schedule(r.grace, func() { r.expireDetach(tab.ID, pending) })
	return result
}

// ReadDetached returns the tab and the move id of a pending detach to the
// first window that reads it, and again to that window only.
func (r *Registry) ReadDetached(windowID, tabID string) (SavedTab, string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pending, ok := r.pendingDetaches[tabID]
	if !ok || windowID == "" || (pending.windowID != "" && pending.windowID != windowID) {
		return SavedTab{}, "", false
	}
	pending.windowID = windowID
	return pending.tab, pending.id, true
}

// detach is a pending detach: the tab with its saved state, the window that
// read it, the id that window passes to Accept, the channel of its result and
// the cancel function of its expiry.
type detach struct {
	id       string
	tab      SavedTab
	windowID string
	result   chan MoveError
	cancel   func() bool
}

func (r *Registry) endDetach(tabID string, pending *detach, outcome MoveError) {
	pending.cancel()
	delete(r.pendingDetaches, tabID)
	pending.result <- outcome
}

// expiry is a pending removal of a window and the cancel func of its run.
// hasAccepted marks a window that accepted a tab while it had no watch.
type expiry struct {
	cancel      func() bool
	hasAccepted bool
}

func (r *Registry) isLive(windowID string) bool {
	if len(r.watches[windowID]) > 0 {
		return true
	}
	pending, ok := r.pendingWindowRemovals[windowID]
	return ok && pending.hasAccepted
}

func (r *Registry) keepAcceptingWindow(windowID string) {
	if len(r.watches[windowID]) > 0 {
		return
	}
	pending, ok := r.pendingWindowRemovals[windowID]
	if !ok {
		pending = &expiry{}
		r.pendingWindowRemovals[windowID] = pending
		pending.cancel = r.schedule(r.grace, func() { r.expireWindow(windowID, pending) })
	}
	pending.hasAccepted = true
}

// CancelDetach cancels the pending detach of a tab and drops its stored state.
// A tab missing from the last report of its window is unlinked.
func (r *Registry) CancelDetach(tabID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pending, ok := r.pendingDetaches[tabID]
	if !ok {
		return
	}
	r.endDetach(tabID, pending, MoveErrorCancelled)
	r.dropUnreportedTab(tabID)
}

func (r *Registry) expireDetach(tabID string, pending *detach) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingDetaches[tabID] != pending {
		return
	}
	r.endDetach(tabID, pending, MoveErrorTimeout)
	r.dropUnreportedTab(tabID)
}

func (r *Registry) dropUnreportedTab(tabID string) {
	tab, isKnown := r.tabs[tabID]
	if !isKnown || slices.Contains(r.windows[tab.WindowID], tabID) {
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

func (r *Registry) removeWindow(windowID string) {
	ids, ok := r.windows[windowID]
	if !ok {
		return
	}
	delete(r.windows, windowID)
	delete(r.windowInfo, windowID)
	r.windowOrder = slices.DeleteFunc(r.windowOrder, func(id string) bool { return id == windowID })
	for _, id := range ids {
		r.forgetTab(id)
	}
	r.publishSnapshot()
}

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

func (r *Registry) getNextColorIndex() int {
	for index := 1; index <= maxColorIndex; index++ {
		if !slices.ContainsFunc(r.groups, func(g *Group) bool { return g.ColorIndex == index }) {
			return index
		}
	}
	return 1
}

func (r *Registry) getSnapshot() Snapshot {
	out := Snapshot{
		Groups:  make([]Group, 0, len(r.groups)),
		Tabs:    make([]Tab, 0, len(r.tabOrder)),
		Windows: r.getWindows(),
	}
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

func (r *Registry) getWindows() []Window {
	out := make([]Window, 0, len(r.windowOrder))
	for _, id := range r.windowOrder {
		if len(r.watches[id]) > 0 {
			out = append(out, r.windowInfo[id])
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
