package flashcards

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// FSRSName is the algorithm. What a schedule is filed under is this and the
// parameters it was worked out on, because the weights are what the numbers
// mean: a build carrying other weights would read another build's stability and
// difficulty as its own, and answer confidently with a wrong day.
const FSRSName = "fsrs-5"

// FSRS spaces a card by how well it came back, carrying two numbers between
// answers: how long the card is expected to stay recalled, and how hard it is.
type FSRS struct {
	engine *fsrs.FSRS
	name   string
}

// NewFSRS is the scheduler on its published parameters.
//
// The fuzz those parameters allow — a day either way, so that cards learnt
// together do not come round together forever — is turned off. A schedule is
// worked out again from the answers whenever it is wanted, and a scheduler that
// answered differently each time would give a different one every launch.
func NewFSRS() FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	return FSRS{engine: fsrs.NewFSRS(p), name: FSRSName + "." + weighed(p)}
}

func (f FSRS) Name() string { return f.name }

// weighed is the parameters as a short name. Everything the scheduler was built
// with goes into it, so parameters that change at all — a weight, a retention, a
// field the library adds — are another name and another cache.
func weighed(p fsrs.Parameters) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%+v", p)))
	return hex.EncodeToString(sum[:4])
}

// Spaced is a card face this scheduler has put into review: it has been
// answered well enough to come round in days.
func (FSRS) Spaced(s Schedule) bool {
	return s.Seen() && fsrs.State(s.Phase) == fsrs.Review
}

func (f FSRS) Next(s Schedule, at time.Time, r Rating) Schedule {
	return left(f.engine.Next(carded(s), at, fsrs.Rating(r)).Card)
}

// Endings is where the two endings leave a card face.
//
// A card face this scheduler has put into review is settled at every rating in
// one reckoning of it, so both endings are asked for together. One it is still
// putting into memory is settled a rating at a time, and is asked a rating at a
// time.
func (f FSRS) Endings(s Schedule, at time.Time) (good, again Schedule) {
	card := carded(s)
	if !f.Spaced(s) {
		return left(f.engine.Next(card, at, fsrs.Good).Card),
			left(f.engine.Next(card, at, fsrs.Again).Card)
	}
	both := f.engine.Repeat(card, at)
	return left(both[fsrs.Good].Card), left(both[fsrs.Again].Card)
}

// carded is a schedule as the library reads a card. One nobody has answered is
// the card that library opens with.
func carded(s Schedule) fsrs.Card {
	if !s.Seen() {
		return fsrs.NewCard()
	}
	return fsrs.Card{
		Due:        s.Due,
		Stability:  s.Stability,
		Difficulty: s.Difficulty,
		Reps:       uint64(s.Reps),
		Lapses:     uint64(s.Lapses),
		State:      fsrs.State(s.Phase),
		LastReview: s.Last,
	}
}

// left is where the library's card stands, as a schedule.
func left(c fsrs.Card) Schedule {
	return Schedule{
		Due:        c.Due,
		Last:       c.LastReview,
		Reps:       int(c.Reps),
		Lapses:     int(c.Lapses),
		Stability:  c.Stability,
		Difficulty: c.Difficulty,
		Phase:      uint8(c.State),
	}
}
