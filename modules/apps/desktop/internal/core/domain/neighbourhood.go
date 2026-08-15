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

// Seated is a note, where it sits, and what the link that seated it says.
type Seated struct {
	NoteRef
	Seat Seat

	// Label is what the person wrote on the link.
	Label string

	// Through is the note the relationship runs from, when that is not the one
	// in focus. A sibling is another child of a shared parent, and which parent
	// is a fact about the vault rather than a choice for whoever draws it.
	Through string
}

// Neighbourhood is one note and everything joined to it, seen from that note.
type Neighbourhood struct {
	Focus   NoteRef
	Related []Seated
}

// Take seats a note. The focus is not related to itself.
func (n *Neighbourhood) Take(note NoteRef, seated Seated) {
	if note.Path == n.Focus.Path {
		return
	}
	seated.NoteRef = note
	n.Related = append(n.Related, seated)
}
