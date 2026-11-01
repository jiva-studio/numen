package domain

// Relation is the position of a note relative to the focus note.
//
// It is not a link role. A role is written in a file and states what one link is
// for; a relation is derived from both ends of every link at once, and one of
// them, the sibling, is written nowhere.
type Relation string

const (
	RelationParent  Relation = "parent"
	RelationChild   Relation = "child"
	RelationJump    Relation = "jump"
	RelationSibling Relation = "sibling"
)

// GetRelationRank orders the relations a note can have. A note can answer to
// two of them — a pair that are each other's parent, a jump between siblings —
// and is shown under only one, so it takes the first it qualifies for.
func GetRelationRank(s Relation) int {
	switch s {
	case RelationParent:
		return 0
	case RelationChild:
		return 1
	case RelationJump:
		return 2
	case RelationSibling:
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

// Neighbour is a note and its relation to the note in focus. The
// records that join it to other notes are in Neighbourhood.Connections.
type Neighbour struct {
	NoteRef
	Relation Relation
}

// Neighbourhood is one note and everything joined to it, seen from that note.
type Neighbourhood struct {
	Focus   NoteRef
	Related []Neighbour

	// Connections holds one connection for every pair of drawn notes that a
	// record joins: the focus with a neighbour, and two neighbours with each
	// other. A hierarchical connection runs from the parent (A) to its child
	// (B). A connection of jumps runs from the later drawn note (A) to the
	// earlier one (B). Records that contradict the direction or the kind of
	// the connection are left out.
	Connections []Connection
}

// AddNeighbour adds a note under its relation. The focus is not related to itself.
func (n *Neighbourhood) AddNeighbour(note NoteRef, neighbour Neighbour) {
	if note.Path == n.Focus.Path {
		return
	}
	neighbour.NoteRef = note
	n.Related = append(n.Related, neighbour)
}
