package review

import (
	"bytes"
	"encoding/json"
	"time"
)

// Rating is how well a card was recalled, one of four values. The user selects
// the rating; the scheduler requires it to space reviews.
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

// Valid reports whether a rating is one of the four. Any other value read from a
// file is an unusable line.
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

// ReviewCardID is the key a schedule is filed under: one card, and one review card of
// the card model that cuts it. The card is identified by its card id, and the review card is
// identified by its name in the card model.
//
// A card is shown once through each of its card model's templates, and each
// template presents a different prompt, so each has its own schedule. The card id
// moves with the card between decks and between vaults; the template name does
// not, and renaming a review card restarts its schedule.
type ReviewCardID struct {
	Card     string
	Template string
}

// Answer is one card answered once, or one answer revoked.
//
// It carries its own identifier, so a revocation references it exactly and
// two answers given in the same millisecond remain distinct.
type Answer struct {
	ID string
	// ReviewCard is which card was shown, through which review card. It is empty on an
	// answer that takes another back.
	ReviewCard ReviewCardID
	At         time.Time
	// Rating is how well the card came back. It is zero on an answer that takes
	// another back.
	Rating Rating
	// Took is how long the card was displayed, recorded to the millisecond. No
	// schedule uses it today; it cannot be measured later.
	Took time.Duration
	// Undoes is the identifier of the answer this one takes back, and is empty
	// on an answer of a card.
	Undoes string
}

// IsUndo reports whether this line takes an answer back and not giving
// one.
func (a Answer) IsUndo() bool { return a.Undoes != "" }

// Version is the format version of one line. A build that reads a line of an
// unknown version skips it, because a partially understood answer counted into a
// schedule is worse than a skipped one.
const Version = 1

// TimestampLayout is how an instant is written: UTC to the millisecond, so
// the lines of one run sort as text in write order. It carries no offset, so a
// time formatted with it must already be in UTC.
const TimestampLayout = "2006-01-02T15:04:05.000Z"

// ParseTimestamp reads an instant as a line carries it, including the offset.
// The file is text in the user's folder and is copied between machines, so any
// RFC 3339 offset is accepted.
func ParseTimestamp(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }

// line is one line of a log, currently in the file.
type line struct {
	V        int    `json:"v"`
	ID       string `json:"id"`
	Card     string `json:"card,omitempty"`
	Template string `json:"template,omitempty"`
	At       string `json:"at"`
	Rating   uint8  `json:"rating,omitempty"`
	Ms       int64  `json:"ms,omitempty"`
	Undo     string `json:"undo,omitempty"`
}

// Write encodes an answer as the bytes stored in a file, including the newline.
//
// A line is written whole or not at all, and the trailing newline marks it as
// complete: any content after the last newline in a file is a run that stopped
// partway.
func Write(a Answer) ([]byte, error) {
	l := line{V: Version, ID: a.ID, At: a.At.UTC().Format(TimestampLayout)}
	if a.IsUndo() {
		l.Undo = a.Undoes
	} else {
		l.Card, l.Template = a.ReviewCard.Card, a.ReviewCard.Template
		l.Rating = uint8(a.Rating)
		l.Ms = a.Took.Milliseconds()
	}
	raw, err := json.Marshal(l)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// Read returns the answers in one log file and the number of lines it could
// not use.
//
// It never fails. A log is appended to by a process that may stop at any moment
// and is copied between machines by external tools, so an unreadable line is
// skipped and counted, and never prevents reading the rest of the history.
func Read(raw []byte) (answers []Answer, skipped int) {
	// Content after the last newline is a line that was not written whole.
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
	if l.Card == "" || l.Template == "" || !Rating(l.Rating).Valid() {
		return Answer{}, false
	}
	return Answer{
		ID:         l.ID,
		ReviewCard: ReviewCardID{Card: l.Card, Template: l.Template},
		At:         pos,
		Rating:     Rating(l.Rating),
		Took:       time.Duration(l.Ms) * time.Millisecond,
	}, true
}
