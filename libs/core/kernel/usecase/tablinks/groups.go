package tablinks

import (
	"slices"
	"strconv"
)

// maxColorIndex is the number of group colors; groups past it share colors.
const maxColorIndex = 4

// groupSet is the link groups of the tabs.
type groupSet struct {
	groups []*Group
	nextID int
}

func (g *groupSet) find(tabID string) *Group {
	for _, group := range g.groups {
		if slices.Contains(group.TabIDs, tabID) {
			return group
		}
	}
	return nil
}

func (g *groupSet) remove(tabID string) bool {
	group := g.find(tabID)
	if group == nil {
		return false
	}
	group.TabIDs = slices.DeleteFunc(group.TabIDs, func(id string) bool { return id == tabID })
	if len(group.TabIDs) < 2 {
		g.groups = slices.DeleteFunc(g.groups, func(one *Group) bool { return one == group })
	}
	return true
}

func (g *groupSet) join(sourceID, targetID string) bool {
	source, target := g.find(sourceID), g.find(targetID)
	switch {
	case source != nil && source == target:
		return false
	case source != nil && target != nil:
		source.TabIDs = appendMissing(source.TabIDs, target.TabIDs...)
		if source.Path == "" {
			source.Path, source.Title = target.Path, target.Title
		}
		g.groups = slices.DeleteFunc(g.groups, func(one *Group) bool { return one == target })
	case source != nil:
		source.TabIDs = append(source.TabIDs, targetID)
	case target != nil:
		target.TabIDs = append(target.TabIDs, sourceID)
	default:
		g.nextID++
		g.groups = append(g.groups, &Group{
			ID:         "group-" + strconv.Itoa(g.nextID),
			ColorIndex: g.getNextColorIndex(),
			TabIDs:     []string{sourceID, targetID},
		})
	}
	return true
}

func (g *groupSet) getNextColorIndex() int {
	for index := 1; index <= maxColorIndex; index++ {
		if !slices.ContainsFunc(g.groups, func(one *Group) bool { return one.ColorIndex == index }) {
			return index
		}
	}
	return 1
}

func (g *groupSet) snapshot() []Group {
	out := make([]Group, 0, len(g.groups))
	for _, group := range g.groups {
		copied := *group
		copied.TabIDs = slices.Clone(group.TabIDs)
		out = append(out, copied)
	}
	return out
}
