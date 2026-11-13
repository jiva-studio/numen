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
	links noteport.LinkBatches
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
// The links are read in one call per step of the walk, and the notes are queried
// once, after every link has been followed.
func NewListRelatedNotes(links noteport.LinkBatches, notes RefQueries) ListRelatedNotes {
	return ListRelatedNotes{links: links, notes: notes}
}

// Execute gathers the notes related to the note at the path. At DepthFar it also gathers,
// for each related note, the notes that one is related to, in the order of the related
// notes: never as siblings, never the focus, the related note itself or a note that is
// already related to the focus.
func (u ListRelatedNotes) Execute(ctx context.Context, v domain.Vault, path string, depth domain.RelationDepth) (domain.NoteGraph, error) {
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
	var relatedToRelated []domain.RelatedNote
	if depth == domain.DepthFar {
		relatedToRelated, err = u.getRelatedToRelated(ctx, cache, path, related)
		if err != nil {
			return domain.NoteGraph{}, err
		}
	}

	notes, err := u.notes.GetNotes(ctx, v.ID, getPaths(path, related, relatedToRelated))
	if err != nil {
		return domain.NoteGraph{}, err
	}
	out, visible := createGraph(notes, path, related, relatedToRelated)
	if len(visible) == 0 {
		return out, nil
	}
	out.Connections, err = getConnections(ctx, cache, visible)
	if err != nil {
		return domain.NoteGraph{}, err
	}
	return out, nil
}

// getPaths lists each path once, the focus first.
func getPaths(focus string, related, relatedToRelated []domain.RelatedNote) []string {
	paths := []string{focus}
	seen := map[string]bool{focus: true}
	for _, n := range slices.Concat(related, relatedToRelated) {
		if !seen[n.Path] {
			seen[n.Path] = true
			paths = append(paths, n.Path)
		}
	}
	return paths
}

// createGraph keeps the notes the index knows. A note related to a related note stays
// only while that related note does. The paths drawn with the focus are returned too.
func createGraph(
	notes map[string]domain.NoteRef, path string, related, relatedToRelated []domain.RelatedNote,
) (domain.NoteGraph, []string) {
	focus, ok := notes[path]
	if !ok {
		return domain.NoteGraph{}, nil
	}
	out := domain.NoteGraph{Focus: focus}
	visible := []string{path}
	for _, s := range related {
		if note, known := notes[s.Path]; known {
			out.AddRelatedNote(note, s)
			visible = append(visible, s.Path)
		}
	}
	for _, item := range relatedToRelated {
		note, known := notes[item.Path]
		if _, ownerShown := notes[item.RelatedTo]; known && ownerShown {
			out.AddRelatedNote(note, item)
		}
	}
	return out, visible
}

func (u ListRelatedNotes) getRelatedToRelated(ctx context.Context, cache *linkCache, focus string, related []domain.RelatedNote) ([]domain.RelatedNote, error) {
	if err := cache.loadBoth(ctx, getPaths(focus, related, nil)); err != nil {
		return nil, err
	}
	direct := map[string]bool{focus: true}
	for _, n := range related {
		direct[n.Path] = true
	}
	var out []domain.RelatedNote
	for _, owner := range related {
		found, err := u.getRelatedNotes(ctx, cache, owner.Path)
		if err != nil {
			return nil, err
		}
		for _, item := range found.getAll() {
			if direct[item.Path] || item.Path == owner.Path {
				continue
			}
			item.RelatedTo = owner.Path
			out = append(out, item)
		}
	}
	return out, nil
}

func (u ListRelatedNotes) addSiblings(ctx context.Context, cache *linkCache, targetSet *relatedNoteSet) error {
	var parents []string
	for _, n := range targetSet.getAll() {
		if n.Relation == domain.RelationParent {
			parents = append(parents, n.Path)
		}
	}
	if err := cache.loadBoth(ctx, parents); err != nil {
		return err
	}
	for _, parent := range parents {
		theirs, err := u.getRelatedNotes(ctx, cache, parent)
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
				Direction: domain.DirectionOutgoing,
			})
		}
	}

	backlinks, err := cache.getBacklinks(ctx, path)
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
			Direction: domain.DirectionIncoming,
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
// Only a reference keeps the direction it was found with.
type relatedNoteSet struct {
	order      []domain.RelatedNote
	targetPath map[string]int
}

func (s *relatedNoteSet) add(related domain.RelatedNote) {
	if related.Relation != domain.RelationReference {
		related.Direction = ""
	}
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
