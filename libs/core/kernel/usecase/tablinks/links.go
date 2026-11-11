package tablinks

// Links groups tabs and carries the path of a group to its tabs.
type Links struct {
	state *state
}

// Link puts two known tabs in one group. If a tab is already in a group, the
// other tab joins it; two groups merge.
func (l *Links) Link(sourceID, targetID string) {
	s := l.state
	s.mu.Lock()
	defer s.mu.Unlock()
	_, isSourceKnown := s.roster.tabs[sourceID]
	_, isTargetKnown := s.roster.tabs[targetID]
	if sourceID == targetID || !isSourceKnown || !isTargetKnown {
		return
	}
	if s.roster.groups.join(sourceID, targetID) {
		s.publishSnapshot()
	}
}

// Unlink removes a tab from its group.
func (l *Links) Unlink(tabID string) {
	s := l.state
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.roster.groups.remove(tabID) {
		s.publishSnapshot()
	}
}

// SyncPath records the new path of a linked tab as the path of its group and
// sends it to every watching window. It returns whether the tab is in a group.
func (l *Links) SyncPath(sourceTabID, path, title string) bool {
	s := l.state
	s.mu.Lock()
	defer s.mu.Unlock()
	group := s.roster.groups.find(sourceTabID)
	if group == nil {
		return false
	}
	group.Path, group.Title = path, title
	s.events.Publish(Event{Sync: &PathSync{
		GroupID: group.ID, SourceTabID: sourceTabID, Path: path, Title: title,
	}})
	return true
}
