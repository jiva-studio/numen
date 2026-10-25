package review

import (
	"math"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// LeastStability is the least a projected card face is taken to stand at, in
// days, which is a minute.
const LeastStability = 1.0 / (24 * 60)

// Schedule is where the answers so far have left one card face.
//
// Nothing in it is written into a vault. It is worked out from the answers
// every time it is wanted, and thrown away at no cost but the working out.
type Schedule struct {
	// Due is when the card face comes round again.
	Due time.Time
	// Last is when it was answered, and is the zero time until it has been.
	Last time.Time
	// Reps is how many answers it has had, Lapses how many of them were Again.
	Reps   int
	Lapses int
	// Stability and Difficulty are what the scheduler carries between answers:
	// how long the card is expected to stay recalled, and how hard it is.
	Stability  float64
	Difficulty float64
	// Phase is the scheduler's own and means nothing outside it. It is kept so
	// that a schedule can be put back where it was without the answers being
	// read again, and it is read by the scheduler that wrote it.
	Phase uint8
}

// IsNew reports whether this card face has never been answered, which is what
// a person means by a new card.
func (s Schedule) IsNew() bool { return s.Last.IsZero() }

// Scheduler works out where an answer leaves a card face.
//
// It is a port and not a function so that the one thing this application must
// be able to change — how cards are spaced — is changed by putting another
// implementation behind it and reading the answers again.
type Scheduler interface {
	// GetName says which scheduler this is, and which version of it. A schedule
	// worked out by one name is not read by another: the numbers a scheduler
	// carries between answers are its own, and one of them read as another's is
	// a wrong answer given confidently.
	GetName() string

	// Next is where an answer leaves a schedule.
	Next(s Schedule, now time.Time, r Rating) Schedule

	// Endings is where an answer leaves a schedule both ways: the ending it
	// came back on and the ending it did not. It is Next at Good and Next at
	// Again, asked together, because a projection weighs the two and a
	// scheduler settles them from one reckoning of the card.
	Endings(s Schedule, now time.Time) (good, again Schedule)

	// Spaced reports whether a card face standing at this schedule comes round
	// in days. One the scheduler is still putting into memory comes round in
	// minutes.
	//
	// Retention is measured over the answers given to spaced card faces and no
	// others: what a card comes back as after ten minutes says nothing about
	// memory.
	IsSpaced(s Schedule) bool
}

// The forgetting curve a projection reads a stability by.
var (
	recallFactor = fsrs.DefaultParam().Factor
	recallDecay  = fsrs.DefaultParam().Decay
)

// Recall is the share of cards standing at this stability that come back after
// this long away. A card nothing is known about comes back to nobody.
func Recall(away time.Duration, stability float64) float64 {
	if stability <= 0 {
		return 0
	}
	days := math.Max(away.Hours()/24, 0)
	return math.Pow(1+recallFactor*days/stability, recallDecay)
}

// RecallChance is what a projection assumes about coming back: how likely a
// card face standing here is to be recalled when it is asked at this instant.
//
// A projection follows one card down the middle of what it may do, weighing the
// ending where it came back against the ending where it did not, and this is
// the weight. Every figure a run draws is drawn under the assumption it was
// given, so a caller naming one says which.
type RecallChance func(Schedule, time.Time) float64

// GetModelledRecall is the chance the scheduler's own forgetting curve gives a card
// face. A run told no other chance reads this one.
func GetModelledRecall(c Schedule, now time.Time) float64 {
	return Recall(now.Sub(c.Last), c.Stability)
}

// GetFullRecall is a run in which every card face asked comes back.
func GetFullRecall(Schedule, time.Time) float64 { return 1 }
