package review

import (
	"bytes"
	"encoding/json"
	"time"
)

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

// IsUndo reports whether this line takes an answer back and not giving
// one.
func (a Answer) IsUndo() bool { return a.Undoes != "" }

// Version is the shape of one line. A build reading a line of a version it does
// not know skips it: an answer it half-understood, counted into a schedule,
// would be worse than one it left out and said so.
const Version = 1

// TimestampLayout is how an instant is written: UTC to the millisecond, so that the lines
// of one run sort as text in the order they were written. It carries no offset,
// so what is formatted with it is a time already in UTC.
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// ParseTimestamp reads an instant as a line carries it, offset and all. The file is
// text in a person's own folder and travels between machines, so what is read
func ParseTimestamp(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }

// line is one line of a log, currently in the file.
type line struct {
	V      int    `json:"v"`
	ID     string `json:"id"`
	Card   string `json:"card,omitempty"`
	Face   string `json:"face,omitempty"`
	At     string `json:"at"`
	Rating uint8  `json:"rating,omitempty"`
	Ms     int64  `json:"ms,omitempty"`
	Undo   string `json:"undo,omitempty"`
}

// Write turns an answer into the bytes it stands as in a file, newline and all.
//
// A line is written whole or not at all, and the newline is what says it landed
// whole: what follows the last one in a file is a run that stopped partway.
func Write(a Answer) ([]byte, error) {
	l := line{V: Version, ID: a.ID, At: a.At.UTC().Format(TimestampLayout)}
	if a.IsUndo() {
		l.Undo = a.Undoes
	} else {
		l.Card, l.Face = a.CardFace.Card, a.CardFace.Face
		l.Rating = uint8(a.Rating)
		l.Ms = a.Took.Milliseconds()
	}
	raw, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// Read takes the answers out of one log file, and says how many lines it could
// not act on.
//
// It never fails. A log is appended to by a process that may stop at any
// moment and is carried between machines by something that knows nothing about
// it, so a line that cannot be read is a line to leave out and count — never a
// reason to refuse the rest of somebody's history.
func Read(raw []byte) (answers []Answer, skipped int) {
	// What stands after the last newline is a line that did not land whole.
	if lastNewline := bytes.LastIndexByte(raw, '\n'); lastNewline < 0 {
		return nil, whole(raw)
	} else if lastNewline+1 < len(raw) {
		skipped, raw = 1, raw[:lastNewline+1]
	}

	for _, one := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(one)) == 0 {
			continue
		}
		a, ok := answer(one)
		if !ok {
			skipped++
			continue
		}
		answers = append(answers, a)
	}
	return answers, skipped
}

// whole counts a file that holds no newline at all: either nothing, or one line
// that did not land.
func whole(raw []byte) int {
	if len(bytes.TrimSpace(raw)) == 0 {
		return 0
	}
	return 1
}

func answer(raw []byte) (Answer, bool) {
	var l line
	if err := json.Unmarshal(raw, &l); err != nil {
		return Answer{}, false
	}
	if l.V != Version || l.ID == "" {
		return Answer{}, false
	}
	pos, err := ParseTimestamp(l.At)
	if err != nil {
		return Answer{}, false
	}
	if l.Undo != "" {
		return Answer{ID: l.ID, At: pos, Undoes: l.Undo}, true
	}
	if l.Card == "" || l.Face == "" || !Rating(l.Rating).Valid() {
		return Answer{}, false
	}
	return Answer{
		ID:       l.ID,
		CardFace: CardFaceID{Card: l.Card, Face: l.Face},
		At:       pos,
		Rating:   Rating(l.Rating),
		Took:     time.Duration(l.Ms) * time.Millisecond,
	}, true
}
