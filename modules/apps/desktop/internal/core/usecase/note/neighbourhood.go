package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ShowNeighbourhood gathers one note and everything joined to it.
//
// A link is written at one end and answered at both, so a note's parents are
// found partly in what it points at and partly in what points at it. Siblings
// are written nowhere: they are the other children of a shared parent.
type ShowNeighbourhood struct {
	Links port.LinkQueries
	Notes port.NoteQueries
}

func (u ShowNeighbourhood) Execute(ctx context.Context, v domain.Vault, path string) (domain.Neighbourhood, error) {
	var out domain.Neighbourhood

	around, err := u.around(ctx, v, path)
	if err != nil {
		return out, err
	}

	// The other children of every parent, which is the whole of what a sibling
	// is. Asked of each parent in turn, because the question is about them, and
	// the parent it came through is kept with it: that is the note the
	// relationship runs from.
	for _, parent := range around.all() {
		if parent.Seat != domain.SeatParent {
			continue
		}
		theirs, err := u.around(ctx, v, parent.Path)
		if err != nil {
			return out, err
		}
		for _, sibling := range theirs.all() {
			if sibling.Seat != domain.SeatChild {
				continue
			}
			around.take(domain.Seated{
				NoteRef: domain.NoteRef{Path: sibling.Path},
				Seat:    domain.SeatSibling,
				Label:   sibling.Label,
				Through: parent.Path,
			})
		}
	}
	around.drop(path)

	seated := around.all()
	paths := make([]string, 0, len(seated)+1)
	paths = append(paths, path)
	for _, s := range seated {
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
	for _, s := range seated {
		if note, known := notes[s.Path]; known {
			out.Take(note, s)
		}
	}
	return out, nil
}

// around seats everything one note is joined to, from both ends of its links.
//
// Which end a link was written at says nothing about the shape of the graph:
// `parent: B` in A and `child: A` in B are the same edge, so the answer has to
// read the role together with the direction it was found in.
func (u ShowNeighbourhood) around(ctx context.Context, v domain.Vault, path string) (*seating, error) {
	seats := &seating{}

	links, err := u.Links.Links(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	for _, l := range links {
		if l.To == "" || l.ToVault != v.ID {
			// Dangling, or landed in another vault. Neither has a seat here:
			// there is nothing to draw and nowhere to go.
			continue
		}
		if seat, navigable := seatFor(l.Role); navigable {
			seats.take(domain.Seated{
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
		if seat, navigable := seatFor(mirror(l.Role)); navigable {
			seats.take(domain.Seated{
				NoteRef: domain.NoteRef{Path: l.From},
				Seat:    seat,
				Label:   l.Label,
			})
		}
	}
	return seats, nil
}

func seatFor(role domain.LinkRole) (domain.Seat, bool) {
	switch role {
	case domain.RoleParent:
		return domain.SeatParent, true
	case domain.RoleChild:
		return domain.SeatChild, true
	case domain.RoleJump:
		return domain.SeatJump, true
	}
	// A wikilink in prose and an attachment are links, and neither is a place
	// in the hierarchy.
	return "", false
}

func mirror(role domain.LinkRole) domain.LinkRole {
	switch role {
	case domain.RoleParent:
		return domain.RoleChild
	case domain.RoleChild:
		return domain.RoleParent
	}
	return role
}

// seating collects notes in the order they were found: the links this note
// wrote, in the order it wrote them; then the links that point at it, in the
// order the query returns; then the siblings each parent brings. The order is
// part of the answer, because it is what the picture is drawn in — and it is
// total, so the same vault gives the same picture twice.
type seating struct {
	order []domain.Seated
	at    map[string]int
}

// take keeps the highest-ranked seat a note qualifies for, replacing a lesser
// one it was given earlier, so that a pair who are each other's parent is drawn
// once rather than twice and always the same way round.
func (s *seating) take(seated domain.Seated) {
	if s.at == nil {
		s.at = map[string]int{}
	}
	i, taken := s.at[seated.Path]
	if !taken {
		s.at[seated.Path] = len(s.order)
		s.order = append(s.order, seated)
		return
	}
	if domain.SeatRank(seated.Seat) < domain.SeatRank(s.order[i].Seat) {
		s.order[i] = seated
	}
}

func (s *seating) drop(path string) {
	i, taken := s.at[path]
	if !taken {
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

func (s *seating) all() []domain.Seated { return s.order }
