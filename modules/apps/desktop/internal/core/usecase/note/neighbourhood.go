package note

import (
	"context"
	"sort"

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

	seats, err := u.around(ctx, v, path)
	if err != nil {
		return out, err
	}

	// The other children of every parent, which is the whole of what a sibling
	// is. Asked of each parent in turn, because the question is about them.
	for target, seat := range seats {
		if seat != domain.SeatParent {
			continue
		}
		theirs, err := u.around(ctx, v, target)
		if err != nil {
			return out, err
		}
		for sibling, theirSeat := range theirs {
			if theirSeat == domain.SeatChild {
				if _, taken := seats[sibling]; !taken {
					seats[sibling] = domain.SeatSibling
				}
			}
		}
	}
	delete(seats, path)

	paths := make([]string, 0, len(seats)+1)
	paths = append(paths, path)
	for target := range seats {
		paths = append(paths, target)
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
	for target, seat := range seats {
		if note, known := notes[target]; known {
			out.Take(note, seat)
		}
	}

	// A picture is not a set: the same vault must draw the same way twice.
	sort.SliceStable(out.Related, func(i, j int) bool {
		if out.Related[i].Seat != out.Related[j].Seat {
			return domain.SeatRank(out.Related[i].Seat) < domain.SeatRank(out.Related[j].Seat)
		}
		return out.Related[i].Title < out.Related[j].Title
	})
	return out, nil
}

// around seats everything one note is joined to, from both ends of its links.
//
// Which end a link was written at says nothing about the shape of the graph:
// `parent: B` in A and `child: A` in B are the same edge, so the answer has to
// read the role together with the direction it was found in.
func (u ShowNeighbourhood) around(ctx context.Context, v domain.Vault, path string) (map[string]domain.Seat, error) {
	seats := map[string]domain.Seat{}

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
		switch l.Role {
		case domain.RoleParent:
			take(seats, l.To, domain.SeatParent)
		case domain.RoleChild:
			take(seats, l.To, domain.SeatChild)
		case domain.RoleJump:
			take(seats, l.To, domain.SeatJump)
		}
	}

	backlinks, err := u.Links.Backlinks(ctx, v.ID, path)
	if err != nil {
		return nil, err
	}
	for _, l := range backlinks {
		switch l.Role {
		case domain.RoleParent:
			// They call this note their parent, so they are its child.
			take(seats, l.From, domain.SeatChild)
		case domain.RoleChild:
			take(seats, l.From, domain.SeatParent)
		case domain.RoleJump:
			take(seats, l.From, domain.SeatJump)
		}
	}
	return seats, nil
}

func take(seats map[string]domain.Seat, path string, seat domain.Seat) {
	if already, taken := seats[path]; !taken || domain.SeatRank(seat) < domain.SeatRank(already) {
		seats[path] = seat
	}
}
