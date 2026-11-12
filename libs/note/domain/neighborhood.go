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
	ID    string
}

// Neighbor is a note and its relation to the note in focus. The
// records that join it to other notes are in Neighborhood.Connections.
type Neighbor struct {
	NoteRef
	Relation Relation
}

// Neighborhood is one note and everything joined to it, seen from that note.
type Neighborhood struct {
	Focus   NoteRef
	Related []Neighbor

	Connections []Connection
}

// AddNeighbor adds a note under its relation. The focus is not related to itself.
func (n *Neighborhood) AddNeighbor(note NoteRef, neighbor Neighbor) {
	if note.Path == n.Focus.Path {
		return
	}
	neighbor.NoteRef = note
	n.Related = append(n.Related, neighbor)
}
