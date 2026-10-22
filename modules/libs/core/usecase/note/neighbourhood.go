package note

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// ShowNeighbourhood gathers one note and everything joined to it.
//
// A link is written at one end and answered at both, so a note's parents are
// found partly in what it points at and partly in what points at it. Siblings
// are written nowhere: they are the other children of a shared parent.
type ShowNeighbourhood struct {
	links port.LinkQueries
	notes RefQueries
}

// RefQueries is the one question a neighbourhood asks about the notes it has
// found: what is needed to show each of them.
type RefQueries interface {
	// GetNotes returns what is needed to show a note, for the paths asked about.
	// Paths that name nothing are absent from the answer: a link resolves as
	// of now, and what it resolved to a moment ago may be gone.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// NewShowNeighbourhood is what a picture of one note is drawn from: where its
// links land, and what is needed to show each note they reach.
//
// The notes are asked for once, after every link has been followed.
func NewShowNeighbourhood(links port.LinkQueries, notes RefQueries) ShowNeighbourhood {
	return ShowNeighbourhood{links: links, notes: notes}
}

func (u ShowNeighbourhood) Execute(ctx context.Context, v domain.Vault, path string) (domain.Neighbourhood, error) {
	center, err := u.getNeighbours(ctx, v, path)
	if err != nil {
		return domain.Neighbourhood{}, err
	}

	if err := u.addSiblings(ctx, v, center); err != nil {
		return domain.Neighbourhood{}, err
	}
	center.remove(path)

	neighbour := center.getAll()
	paths := make([]string, 0, len(neighbour)+1)
	paths = append(paths, path)
	for _, s := range neighbour {
		paths = append(paths, s.Path)
	}
	notes, err := u.notes.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return domain.Neighbourhood{}, err
	}

	focus, recognized := notes[path]
	if !recognized {
		return domain.Neighbourhood{}, nil
	}
	out := domain.Neighbourhood{Focus: focus}
	for _, s := range neighbour {
		if note, recognizedVal := notes[s.Path]; recognizedVal {
			out.AddNeighbour(note, s)
		}
	}
	return out, nil
}

// addSiblings adds the other children of every parent, which is the whole of
// what a sibling is. Each parent is asked in turn, and the sibling keeps the
// parent it came through: that is the note the relationship runs from.
func (u ShowNeighbourhood) addSiblings(ctx context.Context, v domain.Vault, targetSet *neighbourSet) error {
	for _, parent := range targetSet.getAll() {
		if parent.Relation != domain.RelationParent {
			continue
		}
		theirs, err := u.getNeighbours(ctx, v, parent.Path)
		if err != nil {
			return err
		}
		for _, sibling := range theirs.getAll() {
			if sibling.Relation != domain.RelationChild {
				continue
			}
			targetSet.add(domain.Neighbour{
				NoteRef:  domain.NoteRef{Path: sibling.Path},
				Relation: domain.RelationSibling,
				Label:    sibling.Label,
				Parent:   parent.Path,
			})
		}
	}
	return nil
}

// getNeighbours returns everything one note is joined to, from both ends of its links.
//
// Which end a link was written at says nothing about the shape of the graph:
// `parent: B` in A and `child: A` in B are the same edge, so the answer has to
// read the role together with the direction it was found in.
func (u ShowNeighbourhood) getNeighbours(ctx context.Context, v domain.Vault, path string) (*neighbourSet, error) {
	found := &neighbourSet{}

	links, err := u.links.GetLinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	// The notes this one names. One that names it back makes the edge mutual,
	// and a mutual edge takes its label from this end.
	named := map[string]bool{}
	for _, l := range links {
		if l.ToVault != v.ID {
			// A link that is dangling lands in no vault, and one that landed in
			// another vault has no relation here: there is nothing to draw and nowhere
			// to go.
			continue
		}
		if relation, navigable := getRelation(l.Role); navigable {
			named[l.To] = true
			found.add(domain.Neighbour{
				NoteRef:  domain.NoteRef{Path: l.To},
				Relation: relation,
				Label:    l.Label,
			})
		}
	}

	backlinks, err := u.links.GetBacklinks(ctx, v.ID, path)
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
		neighbour := domain.Neighbour{
			NoteRef:  domain.NoteRef{Path: l.From},
			Relation: relation,
			Label:    l.Label,
		}
		if named[l.From] {
			found.addMutual(neighbour)
			continue
		}
		found.add(neighbour)
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

// neighbourSet collects notes in the order they were found: the links this note
// wrote, in the order it wrote them; then the links that point at it, in the
// order the query returns; then the siblings each parent brings. The order is
// part of the answer, because it is what the picture is drawn in — and it is
// total, so the same vault gives the same picture twice.
type neighbourSet struct {
	order      []domain.Neighbour
	targetPath map[string]int
}

// add records a neighbour under the highest-ranked relation it qualifies for,
// replacing a lesser one recorded earlier: a pair who are each other's parent
// is drawn once, and always the same way round.
func (s *neighbourSet) add(neighbour domain.Neighbour) {
	if s.targetPath == nil {
		s.targetPath = map[string]int{}
	}
	i, exists := s.targetPath[neighbour.Path]
	if !exists {
		s.targetPath[neighbour.Path] = len(s.order)
		s.order = append(s.order, neighbour)
		return
	}
	if domain.GetRelationRank(neighbour.Relation) < domain.GetRelationRank(s.order[i].Relation) {
		s.order[i] = neighbour
	}
}

// addMutual records a neighbour that the note in focus names too, and that is
// related already. Both ends naming the same relation is one relationship named
// twice, and the word for it is the focus's own where it wrote one.
func (s *neighbourSet) addMutual(neighbour domain.Neighbour) {
	i := s.targetPath[neighbour.Path]
	current := s.order[i]
	if current.Relation != neighbour.Relation {
		s.add(neighbour)
		return
	}
	current.IsMutual = true
	current.ReverseLabel = neighbour.Label
	if current.Label == "" {
		current.Label = neighbour.Label
	}
	s.order[i] = current
}

// remove takes a note out of the order the picture is drawn in. Nothing is added to
// the relations after it.
func (s *neighbourSet) remove(path string) {
	s.order = slices.DeleteFunc(s.order, func(n domain.Neighbour) bool { return n.Path == path })
}

func (s *neighbourSet) getAll() []domain.Neighbour { return s.order }
