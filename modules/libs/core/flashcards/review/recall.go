package review

import (
	"math"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

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
func GetModelledRecall(c Schedule, at time.Time) float64 {
	return Recall(at.Sub(c.Last), c.Stability)
}

// GetFullRecall is a run in which every card face asked comes back.
func GetFullRecall(Schedule, time.Time) float64 { return 1 }
