package query

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// ShowNeighborhood gathers one note and everything joined to it.
//
// A link is written at one end and answered at both, so a note's parents are
// found partly in what it points at and partly in what points at it. Siblings
// are written nowhere: they are the other children of a shared parent.
type ShowNeighborhood struct {
	links noteport.LinkQueries
	notes RefQueries
}

// RefQueries is the single query a neighborhood makes about the notes it found:
// what is needed to display each of them.
type RefQueries interface {
	// GetNotes returns what is needed to display a note, for the given paths. Paths
	// that match nothing are absent from the result: a link resolves at call time, and
	// the note a link resolved to earlier can be missing.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// NewShowNeighborhood builds the neighborhood of one note: where its links lead,
// and what is needed to display each linked note.
//
// The notes are queried once, after every link has been followed.
func NewShowNeighborhood(links noteport.LinkQueries, notes RefQueries) ShowNeighborhood {
	return ShowNeighborhood{links: links, notes: notes}
}

func (u ShowNeighborhood) Execute(ctx context.Context, v domain.Vault, path string) (domain.Neighborhood, error) {
	cache := &linkCache{queries: u.links, vault: v}
	center, err := u.getNeighbors(ctx, cache, path)
	if err != nil {
		return domain.Neighborhood{}, err
	}

	if err := u.addSiblings(ctx, cache, center); err != nil {
		return domain.Neighborhood{}, err
	}
	center.remove(path)

	neighbor := center.getAll()
	paths := make([]string, 0, len(neighbor)+1)
	paths = append(paths, path)
	for _, s := range neighbor {
		paths = append(paths, s.Path)
	}
	notes, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return domain.Neighborhood{}, err
	}

	focus, recognized := notes[path]
	if !recognized {
		return domain.Neighborhood{}, nil
	}
	out := domain.Neighborhood{Focus: focus}
	visible := []string{path}
	for _, s := range neighbor {
		if note, recognizedVal := notes[s.Path]; recognizedVal {
			out.AddNeighbor(note, s)
			visible = append(visible, s.Path)
		}
	}
	out.Connections, err = getConnections(ctx, cache, visible)
	if err != nil {
		return domain.Neighborhood{}, err
	}
	return out, nil
}

func (u ShowNeighborhood) addSiblings(ctx context.Context, cache *linkCache, targetSet *neighborSet) error {
	for _, parent := range targetSet.getAll() {
		if parent.Relation != domain.RelationParent {
			continue
		}
		theirs, err := u.getNeighbors(ctx, cache, parent.Path)
		if err != nil {
			return err
		}
		for _, sibling := range theirs.getAll() {
			if sibling.Relation != domain.RelationChild {
				continue
			}
			targetSet.add(domain.Neighbor{
				NoteRef:  domain.NoteRef{Path: sibling.Path},
				Relation: domain.RelationSibling,
			})
		}
	}
	return nil
}

func (u ShowNeighborhood) getNeighbors(ctx context.Context, cache *linkCache, path string) (*neighborSet, error) {
	found := &neighborSet{}

	links, err := cache.get(ctx, path)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.ToVault != cache.vault.ID {
			// A link that is dangling lands in no vault, and one that landed in
			// another vault has no relation here: there is nothing to draw and nowhere
			// to go.
			continue
		}
		if relation, navigable := getRelation(l.Role); navigable {
			found.add(domain.Neighbor{
				NoteRef:  domain.NoteRef{Path: l.To},
				Relation: relation,
			})
		}
	}

	backlinks, err := u.links.GetBacklinks(ctx, cache.vault.ID, path)
	if err != nil {
		return nil, err
	}
	for _, l := range backlinks {
		// The role is read from the other end, so it means the opposite: a note
		// that calls this one its parent is its child.
		relation, navigable := getRelation(getOppositeRole(l.Role))
		if !navigable {
			continue
		}
		found.add(domain.Neighbor{
			NoteRef:  domain.NoteRef{Path: l.From},
			Relation: relation,
		})
	}
	return found, nil
}

func getRelation(role domain.LinkRole) (domain.Relation, bool) {
	switch role {
	case domain.RoleParent:
		return domain.RelationParent, true
	case domain.RoleChild:
		return domain.RelationChild, true
	case domain.RoleJump:
		return domain.RelationJump, true
	default:
		// A wikilink in prose and an attachment are links, and neither is a
		// place in the hierarchy.
		return "", false
	}
}

func getOppositeRole(role domain.LinkRole) domain.LinkRole {
	switch role {
	case domain.RoleParent:
		return domain.RoleChild
	case domain.RoleChild:
		return domain.RoleParent
	default:
		return role
	}
}

// neighborSet collects notes in discovery order: the links this note declares, in
// declaration order; then the links that point at it, in query order; then the
// siblings of each parent. The order is part of the result, because it is the
// drawing order, and it is total, so the same vault always gives the same layout.
type neighborSet struct {
	order      []domain.Neighbor
	targetPath map[string]int
}

func (s *neighborSet) add(neighbor domain.Neighbor) {
	if s.targetPath == nil {
		s.targetPath = map[string]int{}
	}
	i, exists := s.targetPath[neighbor.Path]
	if !exists {
		s.targetPath[neighbor.Path] = len(s.order)
		s.order = append(s.order, neighbor)
		return
	}
	if domain.GetRelationRank(neighbor.Relation) < domain.GetRelationRank(s.order[i].Relation) {
		s.order[i] = neighbor
	}
}

func (s *neighborSet) remove(path string) {
	s.order = slices.DeleteFunc(s.order, func(n domain.Neighbor) bool { return n.Path == path })
}

func (s *neighborSet) getAll() []domain.Neighbor { return s.order }
