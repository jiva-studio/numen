package review

import "time"

// Rating is how well a card came back. A person says which of the four, and
// nothing here decides it for them: how well a thing was recalled is what a
// scheduler has to be told to space anything sensibly.
type Rating uint8

const (
	// Again is a card that did not come back at all.
	Again Rating = iota + 1
	// Hard is a card that came back, slowly and with effort.
	Hard
	// Good is a card that came back.
	Good
	// Easy is a card that came back with none.
	Easy
)

// Valid reports whether a rating is one of the four. Anything else read out of
// a file is a line nobody can act on.
func (r Rating) Valid() bool { return r >= Again && r <= Easy }

func (r Rating) String() string {
	switch r {
	case Again:
		return "again"
	case Hard:
		return "hard"
	case Good:
		return "good"
	case Easy:
		return "easy"
	}
	return "unknown"
}

// CardFaceID is the key a schedule is filed under: one card, and one face of
// the stencil that cuts it. The card is the mark it is known by, and the face is
// the name it carries in its stencil.
//
// A card is shown once through each of its stencil's faces, and each of them
// asks a different thing, so each has a path of its own. The mark travels with
// the card between decks and between vaults; the face's name does not travel at
// all, and renaming a face starts its schedule again.
type CardFaceID struct {
	Card string
	Face string
}

// Answer is one card answered once, or one answer taken back.
//
// It carries an identifier of its own so that taking it back names it exactly,
// and so that two answers given in the same millisecond stay two.
type Answer struct {
	ID string
	// CardFace is which card was shown, through which face. It is empty on an
	// answer that takes another back.
	CardFace CardFaceID
	At       time.Time
	// Rating is how well the card came back. It is zero on an answer that takes
	// another back.
	Rating Rating
	// Took is how long the card stood on the screen, recorded to the
	// millisecond. Nothing schedules by it today; it is the one thing about an
	// answer that cannot be measured later.
	Took time.Duration
	// Undoes is the identifier of the answer this one takes back, and is empty
	// on an answer of a card.
	Undoes string
}

// IsUndo reports whether this line takes an answer back rather than giving
// one.
func (a Answer) IsUndo() bool { return a.Undoes != "" }
