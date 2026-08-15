package domain

// Seat is where a note sits relative to the one being looked at.
//
// It is not a link role. A role is written in a file and says what one link is
// for; a seat is worked out from both ends of every link at once, and one of
// them — the sibling — is written nowhere at all.
type Seat string

const (
	SeatParent  Seat = "parent"
	SeatChild   Seat = "child"
	SeatJump    Seat = "jump"
	SeatSibling Seat = "sibling"
)

// SeatRank orders the seats a note can take. A note can answer to two of them —
// a pair that are each other's parent, a jump between siblings — and can be
// shown in only one place, so it takes the first it qualifies for.
func SeatRank(s Seat) int {
	switch s {
	case SeatParent:
		return 0
	case SeatChild:
		return 1
	case SeatJump:
		return 2
	case SeatSibling:
		return 3
	default:
		return 4
	}
}

// NoteRef is a note as something else refers to it: enough to show it and to
// ask for it again.
type NoteRef struct {
	Path  string
	Title string
	// ID is the identifier written in the file, empty for a note made outside
	// the application.
	ID string
}

// Seated is a note and where it sits.
type Seated struct {
	NoteRef
	Seat Seat
}

// Neighbourhood is one note and everything joined to it, seen from that note.
type Neighbourhood struct {
	Focus   NoteRef
	Related []Seated
}

// Take gives a note a seat, keeping the first one it qualifies for.
func (n *Neighbourhood) Take(note NoteRef, seat Seat) {
	if note.Path == n.Focus.Path {
		return
	}
	for i, already := range n.Related {
		if already.Path != note.Path {
			continue
		}
		if SeatRank(seat) < SeatRank(already.Seat) {
			n.Related[i].Seat = seat
		}
		return
	}
	n.Related = append(n.Related, Seated{NoteRef: note, Seat: seat})
}
