package review

import (
	"math"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

// LeastStability is the least a projected review card is taken to stand at, in
// days, which is a minute.
const LeastStability = 1.0 / (24 * 60)

// Schedule is where the answers so far have left one review card.
//
// Nothing in it is written into a vault. It is worked out from the answers
// every time it is wanted, and thrown away at no cost but the working out.
type Schedule struct {
	// Due is when the review card comes round again.
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

// noSchedule is the schedule of a review card that has not been answered.
var noSchedule = Schedule{
	Due: time.Time{}, Last: time.Time{}, Reps: 0, Lapses: 0, Stability: 0, Difficulty: 0, Phase: 0,
}

// IsNew reports whether this review card has never been answered, which defines
// a new card.
func (s Schedule) IsNew() bool { return s.Last.IsZero() }

// Scheduler computes where an answer leaves a review card.
//
// It is a port because the one behavior this application must be able to change,
// how cards are spaced, is changed by supplying another implementation and
// replaying the answers.
type Scheduler interface {
	// GetName returns the scheduler's name and version. A schedule computed under
	// one name is not read under another: the values a scheduler carries between
	// answers are specific to it, and one scheduler's values read as another's
	// give a wrong result.
	GetName() string

	// Next is where an answer leaves a schedule.
	Next(s Schedule, now time.Time, r Rating) Schedule

	// Endings returns both outcomes of an answer: the schedule after a recalled
	// answer (Next at Good) and after a forgotten one (Next at Again). They are
	// computed together because a projection weighs the two and a scheduler
	// derives them from one evaluation of the card.
	Endings(s Schedule, now time.Time) (good, again Schedule)

	// IsSpaced reports whether a review card at this schedule is next due in
	// days. A card the scheduler is still learning is due in minutes.
	//
	// Retention is measured over the answers given to spaced review cards only: an
	// answer after ten minutes carries no information about memory.
	IsSpaced(s Schedule) bool
}

// The forgetting curve a projection reads a stability by.
var (
	recallFactor = fsrs.DefaultParam().Factor
	recallDecay  = fsrs.DefaultParam().Decay
)

// Recall is the share of cards at this stability that are recalled after this
// long away. A stability of zero or less yields zero.
func Recall(away time.Duration, stability float64) float64 {
	if stability <= 0 {
		return 0
	}
	days := math.Max(away.Hours()/24, 0)
	return math.Pow(1+recallFactor*days/stability, recallDecay)
}

// RecallChance is the recall probability a projection assumes: how likely a
// review card at this schedule is to be recalled when presented at this instant.
//
// A projection follows one card along the central path of what it may do,
// weighing the recalled ending against the forgotten ending by this
// probability. Every figure a run produces depends on the assumption it was
// given, so a caller naming a figure states which.
type RecallChance func(Schedule, time.Time) float64

// GetModelledRecall is the recall probability from the scheduler's own
// forgetting curve for a review card. A run with no other probability uses this one.
func GetModelledRecall(c Schedule, now time.Time) float64 {
	return Recall(now.Sub(c.Last), c.Stability)
}

// GetFullRecall is the recall probability under which every review card is recalled.
func GetFullRecall(Schedule, time.Time) float64 { return 1 }
