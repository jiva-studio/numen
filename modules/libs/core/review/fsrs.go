package review

import (
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// FSRSName is what a schedule this scheduler worked out is filed under.
const FSRSName = "fsrs-5"

// FSRS spaces a card by how well it came back, carrying two numbers between
// answers: how long the card is expected to stay recalled, and how hard it is.
type FSRS struct{ engine *fsrs.FSRS }

// NewFSRS is the scheduler on its published parameters.
//
// The fuzz those parameters allow — a day either way, so that cards learnt
// together do not come round together forever — is turned off. A schedule is
// worked out again from the answers whenever it is wanted, and a scheduler that
// answered differently each time would give a different one every launch.
func NewFSRS() FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	return FSRS{engine: fsrs.NewFSRS(p)}
}

func (FSRS) Name() string { return FSRSName }

func (f FSRS) Next(s Schedule, at time.Time, r Rating) Schedule {
	card := fsrs.NewCard()
	if s.Seen() {
		card = fsrs.Card{
			Due:        s.Due,
			Stability:  s.Stability,
			Difficulty: s.Difficulty,
			Reps:       uint64(s.Reps),
			Lapses:     uint64(s.Lapses),
			State:      fsrs.State(s.Phase),
			LastReview: s.Last,
		}
	}
	out := f.engine.Next(card, at, fsrs.Rating(r)).Card
	return Schedule{
		Due:        out.Due,
		Last:       out.LastReview,
		Reps:       int(out.Reps),
		Lapses:     int(out.Lapses),
		Stability:  out.Stability,
		Difficulty: out.Difficulty,
		Phase:      uint8(out.State),
	}
}
