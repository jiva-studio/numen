package query

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

// ListRelatedNotes gathers the notes related to one note, and the note itself.
//
// A link is written at one end and answered at both, so a note's parents are
// found partly in what it points at and partly in what points at it. Siblings
// are written nowhere: they are the other children of a shared parent.
type ListRelatedNotes struct {
	links noteport.LinkQueries
	notes RefQueries
}

// RefQueries is the single query the related-notes lookup makes about the notes it found:
// what is needed to display each of them.
type RefQueries interface {
	// GetNotes returns what is needed to display a note, for the given paths. Paths
	// that match nothing are absent from the result: a link resolves at call time, and
	// the note a link resolved to earlier can be missing.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// NewListRelatedNotes builds the related notes of one note: where its links lead,
// and what is needed to display each linked note.
//
// The notes are queried once, after every link has been followed.
func NewListRelatedNotes(links noteport.LinkQueries, notes RefQueries) ListRelatedNotes {
	return ListRelatedNotes{links: links, notes: notes}
}

func (u ListRelatedNotes) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteGraph, error) {
	cache := &linkCache{queries: u.links, vault: v}
	center, err := u.getRelatedNotes(ctx, cache, path)
	if err != nil {
		return domain.NoteGraph{}, err
	}

	if err := u.addSiblings(ctx, cache, center); err != nil {
		return domain.NoteGraph{}, err
	}
	center.remove(path)

	related := center.getAll()
	paths := make([]string, 0, len(related)+1)
	paths = append(paths, path)
	for _, s := range related {
		paths = append(paths, s.Path)
	}
	notes, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return domain.NoteGraph{}, err
	}

	focus, recognized := notes[path]
	if !recognized {
		return domain.NoteGraph{}, nil
	}
	out := domain.NoteGraph{Focus: focus}
	visible := []string{path}
	for _, s := range related {
		if note, recognizedVal := notes[s.Path]; recognizedVal {
			out.AddRelatedNote(note, s)
			visible = append(visible, s.Path)
		}
	}
	out.Connections, err = getConnections(ctx, cache, visible)
	if err != nil {
		return domain.NoteGraph{}, err
	}
	return out, nil
}

func (u ListRelatedNotes) addSiblings(ctx context.Context, cache *linkCache, targetSet *relatedNoteSet) error {
	for _, parent := range targetSet.getAll() {
		if parent.Relation != domain.RelationParent {
			continue
		}
		theirs, err := u.getRelatedNotes(ctx, cache, parent.Path)
		if err != nil {
			return err
		}
		for _, sibling := range theirs.getAll() {
			if sibling.Relation != domain.RelationChild {
				continue
			}
			targetSet.add(domain.RelatedNote{
				NoteRef:  domain.NoteRef{Path: sibling.Path},
				Relation: domain.RelationSibling,
			})
		}
	}
	return nil
}

func (u ListRelatedNotes) getRelatedNotes(ctx context.Context, cache *linkCache, path string) (*relatedNoteSet, error) {
	found := &relatedNoteSet{}

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
			found.add(domain.RelatedNote{
				NoteRef:   domain.NoteRef{Path: l.To},
				Relation:  relation,
				Direction: getDirection(relation, domain.DirectionOutgoing),
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
		found.add(domain.RelatedNote{
			NoteRef:   domain.NoteRef{Path: l.From},
			Relation:  relation,
			Direction: getDirection(relation, domain.DirectionIncoming),
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
	case domain.RoleRef:
		return domain.RelationReference, true
	default:
		// An attachment is a link, and not a place in the hierarchy.
		return "", false
	}
}

func getDirection(relation domain.Relation, direction domain.LinkDirection) domain.LinkDirection {
	if relation != domain.RelationReference {
		return ""
	}
	return direction
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

// relatedNoteSet collects notes in discovery order: the links this note declares, in
// declaration order; then the links that point at it, in query order; then the
// siblings of each parent. The order is part of the result, because it is the
// drawing order, and it is total, so the same vault always gives the same layout.
type relatedNoteSet struct {
	order      []domain.RelatedNote
	targetPath map[string]int
}

func (s *relatedNoteSet) add(related domain.RelatedNote) {
	if s.targetPath == nil {
		s.targetPath = map[string]int{}
	}
	i, exists := s.targetPath[related.Path]
	if !exists {
		s.targetPath[related.Path] = len(s.order)
		s.order = append(s.order, related)
		return
	}
	kept := s.order[i]
	if domain.GetRelationRank(related.Relation) < domain.GetRelationRank(kept.Relation) {
		s.order[i] = related
		return
	}
	if kept.Relation == domain.RelationReference && related.Relation == domain.RelationReference &&
		kept.Direction != related.Direction {
		s.order[i].Direction = domain.DirectionBoth
	}
}

func (s *relatedNoteSet) remove(path string) {
	s.order = slices.DeleteFunc(s.order, func(n domain.RelatedNote) bool { return n.Path == path })
}

func (s *relatedNoteSet) getAll() []domain.RelatedNote { return s.order }
