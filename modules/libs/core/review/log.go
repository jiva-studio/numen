package review

import (
	"bytes"
	"encoding/json"
	"time"
)

// Version is the shape of one line. A build reading a line of a version it does
// not know skips it: an answer it half-understood, counted into a schedule,
// would be worse than one it left out and said so.
const Version = 1

// Stamp is how an instant is written: UTC to the millisecond, so that the lines
// of one run sort as text in the order they were written. It carries no offset,
// so what is formatted with it is a time already in UTC.
const Stamp = "2006-01-02T15:04:05.000Z"

// Moment reads an instant as a line carries it, offset and all. The file is
// text in a person's own folder and travels between machines, so what is read
// back is not held to the one shape this build writes.
func Moment(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }

// line is one line of a log, as it stands in the file.
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
	l := line{V: Version, ID: a.ID, At: a.At.UTC().Format(Stamp)}
	if a.TakesBack() {
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
	if cut := bytes.LastIndexByte(raw, '\n'); cut < 0 {
		return nil, whole(raw)
	} else if cut+1 < len(raw) {
		skipped, raw = 1, raw[:cut+1]
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
	at, err := Moment(l.At)
	if err != nil {
		return Answer{}, false
	}
	if l.Undo != "" {
		return Answer{ID: l.ID, At: at, Undoes: l.Undo}, true
	}
	if l.Card == "" || l.Face == "" || !Rating(l.Rating).Valid() {
		return Answer{}, false
	}
	return Answer{
		ID:       l.ID,
		CardFace: CardFaceID{Card: l.Card, Face: l.Face},
		At:       at,
		Rating:   Rating(l.Rating),
		Took:     time.Duration(l.Ms) * time.Millisecond,
	}, true
}
