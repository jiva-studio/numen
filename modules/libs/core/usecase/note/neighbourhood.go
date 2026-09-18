package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ShowNeighbourhood gathers one note and everything joined to it.
//
// A link is written at one end and answered at both, so a note's parents are
// found partly in what it points at and partly in what points at it. Siblings
// are written nowhere: they are the other children of a shared parent.
type ShowNeighbourhood struct {
	Links port.LinkQueries
	Notes RefQueries
}

// RefQueries is the one question a neighbourhood asks about the notes it has
// found: what is needed to show each of them.
type RefQueries interface {
	// Notes returns what is needed to show a note, for the paths asked about.
	// Paths that name nothing are absent from the answer: a link resolves as
	// of now, and what it resolved to a moment ago may be gone.
	Notes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// NewShowNeighbourhood is what a picture of one note is drawn from: where its
// links land, and what is needed to show each note they reach.
//
// The notes are asked for once, after every link has been followed.
func NewShowNeighbourhood(links port.LinkQueries, notes RefQueries) ShowNeighbourhood {
	return ShowNeighbourhood{Links: links, Notes: notes}
}

func (u ShowNeighbourhood) Execute(ctx context.Context, v domain.Vault, path string) (domain.Neighbourhood, error) {
	var out domain.Neighbourhood

	around, err := u.getSeats(ctx, v, path)
	if err != nil {
		return out, err
	}

	// The other children of every parent, which is the whole of what a sibling
	// is. Asked of each parent in turn, because the question is about them, and
	// the parent it came through is kept with it: that is the note the
	// relationship runs from.
	for _, parent := range around.getAll() {
		if parent.Seat != domain.SeatParent {
			continue
		}
		theirs, err := u.getSeats(ctx, v, parent.Path)
		if err != nil {
			return out, err
		}
		for _, sibling := range theirs.getAll() {
			if sibling.Seat != domain.SeatChild {
				continue
			}
			around.add(domain.Neighbour{
				NoteRef: domain.NoteRef{Path: sibling.Path},
				Seat:    domain.SeatSibling,
				Label:   sibling.Label,
				Parent:  parent.Path,
			})
		}
	}
	around.remove(path)

	neighbour := around.getAll()
	paths := make([]string, 0, len(neighbour)+1)
	paths = append(paths, path)
	for _, s := range neighbour {
		paths = append(paths, s.Path)
	}
	notes, err := u.Notes.Notes(ctx, v.ID, paths)
	if err != nil {
		return out, err
	}

	focus, known := notes[path]
	if !known {
		return out, nil
	}
	out.Focus = focus
	for _, s := range neighbour {
		if note, known := notes[s.Path]; known {
			out.AddNeighbour(note, s)
		}
	}
	return out, nil
}

// getSeats seats everything one note is joined to, from both ends of its links.
//
// Which end a link was written at says nothing about the shape of the graph:
// `parent: B` in A and `child: A` in B are the same edge, so the answer has to
// read the role together with the direction it was found in.
func (u ShowNeighbourhood) getSeats(ctx context.Context, v domain.Vault, path string) (*seats, error) {
	seats := &seats{}

	links, err := u.Links.Links(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	// The notes this one names. One that names it back makes the edge mutual,
	// and a mutual edge takes its label from this end.
	named := map[string]bool{}
	for _, l := range links {
		if l.To == "" || l.ToVault != v.ID {
			// Dangling, or landed in another vault. Neither has a seat here:
			// there is nothing to draw and nowhere to go.
			continue
		}
		if seat, navigable := seatFor(l.Role); navigable {
			named[l.To] = true
			seats.add(domain.Neighbour{
				NoteRef: domain.NoteRef{Path: l.To},
				Seat:    seat,
				Label:   l.Label,
			})
		}
	}

	backlinks, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	for _, l := range backlinks {
		// The role is read from the other end, so it means the opposite: a note
		// that calls this one its parent is its child.
		seat, navigable := seatFor(getOppositeRole(l.Role))
		if !navigable {
			continue
		}
		neighbour := domain.Neighbour{
			NoteRef: domain.NoteRef{Path: l.From},
			Seat:    seat,
			Label:   l.Label,
		}
		if named[l.From] {
			seats.addMutual(neighbour)
			continue
		}
		seats.add(neighbour)
	}
	return seats, nil
}

func seatFor(role domain.LinkRole) (domain.Relation, bool) {
	switch role {
	case domain.RoleParent:
		return domain.SeatParent, true
	case domain.RoleChild:
		return domain.SeatChild, true
	case domain.RoleJump:
		return domain.SeatJump, true
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

// seats collects notes in the order they were found: the links this note
// wrote, in the order it wrote them; then the links that point at it, in the
// order the query returns; then the siblings each parent brings. The order is
// part of the answer, because it is what the picture is drawn in — and it is
// total, so the same vault gives the same picture twice.
type seats struct {
	order []domain.Neighbour
	at    map[string]int
}

// add records a neighbour under the highest-ranked relation it qualifies for,
// replacing a lesser one recorded earlier: a pair who are each other's parent
// is drawn once, and always the same way round.
func (s *seats) add(neighbour domain.Neighbour) {
	if s.at == nil {
		s.at = map[string]int{}
	}
	i, exists := s.at[neighbour.Path]
	if !exists {
		s.at[neighbour.Path] = len(s.order)
		s.order = append(s.order, neighbour)
		return
	}
	if domain.SeatRank(neighbour.Seat) < domain.SeatRank(s.order[i].Seat) {
		s.order[i] = neighbour
	}
}

// addMutual records a neighbour that the note in focus names too. Both ends
// naming the same relation is one relationship named twice, and the word for it
// is the focus's own where it wrote one.
func (s *seats) addMutual(neighbour domain.Neighbour) {
	i, exists := s.at[neighbour.Path]
	if !exists {
		s.add(neighbour)
		return
	}
	current := s.order[i]
	if current.Seat != neighbour.Seat {
		s.add(neighbour)
		return
	}
	current.IsMutual = true
	if current.Label == "" {
		current.Label = neighbour.Label
	}
	s.order[i] = current
}

func (s *seats) remove(path string) {
	i, exists := s.at[path]
	if !exists {
		return
	}
	delete(s.at, path)
	s.order = append(s.order[:i], s.order[i+1:]...)
	for p, at := range s.at {
		if at > i {
			s.at[p] = at - 1
		}
	}
}

func (s *seats) getAll() []domain.Neighbour { return s.order }
