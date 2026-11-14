package domain

import (
	"errors"
	"fmt"
)

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

	// RelationReference is a link in the text of one of the two notes.
	RelationReference Relation = "reference"
)

// LinkDirection tells which note wrote a text link. "The note in focus" below is
// the note a related note belongs to: the focus, or RelatedTo at depth two.
type LinkDirection string

const (
	// DirectionOutgoing: the note in focus links to the related note.
	DirectionOutgoing LinkDirection = "outgoing"
	// DirectionIncoming: the related note links to the note in focus.
	DirectionIncoming LinkDirection = "incoming"
	// DirectionBoth: each of the two notes links to the other.
	DirectionBoth LinkDirection = "both"
)

// GetRelationRank orders the relations a note can have: parent, child, jump,
// sibling, reference. A note can answer to
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
	case RelationReference:
		return 4
	default:
		return 5
	}
}

// RelationDepth is how many steps from the focus the related notes are gathered.
type RelationDepth uint8

const (
	// DepthNear gathers the notes related to the focus.
	DepthNear RelationDepth = 1
	// DepthFar also gathers the notes related to each of those.
	DepthFar RelationDepth = 2
)

// ErrInvalidRelationDepth is a depth that is none of the two.
var ErrInvalidRelationDepth = errors.New("the depth of related notes is 1 or 2")

// ParseRelationDepth reads the depth a client sends: 0 means DepthNear.
func ParseRelationDepth(wire uint32) (RelationDepth, error) {
	switch wire {
	case 0, 1:
		return DepthNear, nil
	case 2:
		return DepthFar, nil
	default:
		return 0, fmt.Errorf("%w, not %d", ErrInvalidRelationDepth, wire)
	}
}

// NoteRef is a note as something else refers to it: enough to show it and to
// ask for it again.
type NoteRef struct {
	Path  string
	Title string
	ID    string
}

// RelatedNote is a note and its relation to the note in focus. RelatedTo is the
// path of the related note this one belongs to, set for a note found two steps
// from the focus; Relation and Direction are then relative to that note, not to
// the focus. Empty otherwise. Direction is set for RelationReference only. A
// note two steps away has no connection records, and its path can repeat in a
// graph: once under each related note it belongs to.
type RelatedNote struct {
	NoteRef
	Relation  Relation
	Direction LinkDirection
	RelatedTo string
}

// NoteGraph is one note and everything joined to it, seen from that note.
type NoteGraph struct {
	Focus   NoteRef
	Related []RelatedNote

	Connections []Connection
}

// AddRelatedNote adds a note under its relation. The focus is not related to itself.
func (n *NoteGraph) AddRelatedNote(note NoteRef, related RelatedNote) {
	if note.Path == n.Focus.Path {
		return
	}
	related.NoteRef = note
	n.Related = append(n.Related, related)
}
