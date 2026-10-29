package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"math"
	"strconv"
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
//
// The parameters come from the library and the arithmetic over them is here, so a
// scheduler carries no state between answers and one instance can serve many
// review cards concurrently.
type FSRS struct {
	p    fsrs.Parameters
	name string
}

// NewFSRS is the scheduler on its published parameters.
//
// Fuzz, the library's random offset of a day either way that keeps cards learned
// together from falling due together, is disabled. A schedule is recomputed from
// the answers whenever it is needed, and a scheduler with varying output would
// produce a different schedule on every launch.
func NewFSRS() FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	return FSRS{p: p, name: FSRSName + "." + hashParameters(p)}
}

// NewFSRSAt is the scheduler asking for this share of the cards to come back
// when they come round. A share outside what a preset may hold is brought to
// the nearest end of it.
func NewFSRSAt(retention float64) FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	p.RequestRetention = math.Min(math.Max(retention, RetentionBounds.Min), RetentionBounds.Max)
	return FSRS{p: p, name: FSRSName + "." + hashParameters(p)}
}

// GetName is the name of the scheduler, which includes a hash of its parameters.
func (f FSRS) GetName() string { return f.name }

// hashParameters is the parameters as a short name. Every number the
// arithmetic above reads goes into it under a name of its own, so a weight or a
// retention that changes is another name and another cache, and a field the
// library adds, renames or reorders is not.
//
// The numbers go in as hexadecimal floats, which are exact and are written the
// same way at every release.
func hashParameters(p fsrs.Parameters) string {
	sum := sha256.New()
	for _, one := range []struct {
		name  string
		value float64
	}{
		{"retention", p.RequestRetention},
		{"interval", p.MaximumInterval},
		{"decay", p.Decay},
		{"factor", p.Factor},
	} {
		writeNumber(sum, one.name, one.value)
	}
	for i, w := range p.W {
		writeNumber(sum, "w"+strconv.Itoa(i), w)
	}
	return hex.EncodeToString(sum.Sum(nil)[:4])
}

// writeNumber writes one number of the parameters into the name being worked
// out.
func writeNumber(sum hash.Hash, name string, value float64) {
	fmt.Fprintf(sum, "%s=%s\n", name, strconv.FormatFloat(value, 'x', -1, 64))
}

// IsSpaced reports whether this scheduler has put a review card into review: it
// has been answered well enough to come round in days.
func (FSRS) IsSpaced(s Schedule) bool {
	return !s.IsNew() && fsrs.State(s.Phase) == fsrs.Review
}

// Next is where an answer leaves a schedule.
func (f FSRS) Next(s Schedule, now time.Time, r Rating) Schedule {
	one := f.readAtAnswer(s, now)
	switch one.phase {
	case fsrs.New:
		return f.getNextFromNew(one, now, fsrs.Rating(r))
	case fsrs.Review:
		return f.getNextFromReview(one, now, f.getRecall(one), fsrs.Rating(r))
	case fsrs.Learning, fsrs.Relearning:
		return f.getNextFromLearning(one, now, fsrs.Rating(r))
	default:
		return f.getNextFromLearning(one, now, fsrs.Rating(r))
	}
}

// Endings is where the two endings leave a review card.
//
// A review card this scheduler has put into review is settled at every rating by
// one reckoning of how likely it was to come back, so both endings are worked
// out from that one reckoning.
func (f FSRS) Endings(s Schedule, now time.Time) (good, again Schedule) {
	one := f.readAtAnswer(s, now)
	switch one.phase {
	case fsrs.New:
		return f.getNextFromNew(one, now, fsrs.Good), f.getNextFromNew(one, now, fsrs.Again)
	case fsrs.Review:
		back := f.getRecall(one)
		return f.getNextFromReview(one, now, back, fsrs.Good),
			f.getNextFromReview(one, now, back, fsrs.Again)
	case fsrs.Learning, fsrs.Relearning:
		return f.getNextFromLearning(one, now, fsrs.Good),
			f.getNextFromLearning(one, now, fsrs.Again)
	default:
		return f.getNextFromLearning(one, now, fsrs.Good),
			f.getNextFromLearning(one, now, fsrs.Again)
	}
}

// atAnswer is a review card at the instant it is answered: its phase, the whole
// days elapsed since its last answer, and the schedule after the answer for
// everything no rating decides.
type atAnswer struct {
	phase fsrs.State
	// away is the number of whole days elapsed since the last answer, and zero for a
	// card that has not been answered.
	away float64
	// last is the schedule before the answer, and out is the schedule after the
	// answer except for the fields the rating decides.
	last Schedule
	out  Schedule
}

// readAtAnswer builds the atAnswer state of a review card at the instant it is
// answered. An unanswered card starts in the phase the scheduler begins cards in,
// with zero stability and zero difficulty.
func (f FSRS) readAtAnswer(s Schedule, pos time.Time) atAnswer {
	one := atAnswer{phase: fsrs.New, away: 0, last: noSchedule, out: noSchedule}
	if !s.IsNew() {
		one.last, one.phase = s, fsrs.State(s.Phase)
	}
	// A review card in review carries a positive stability, read as a divisor
	// and as the base of a power in getRecall and getStabilityOnRecall; zero
	// or negative is floored to 0.1, the lowest this scheduler itself opens
	// one at.
	if one.phase == fsrs.Review && one.last.Stability <= 0 {
		one.last.Stability = 0.1
	}
	// The days away are the whole days gone by, and they do not go below none.
	if one.phase != fsrs.New {
		one.away = math.Max(math.Floor(pos.Sub(one.last.Last).Hours()/24), 0)
	}
	one.out = one.last
	one.out.Last = pos
	one.out.Reps = one.last.Reps + 1
	return one
}

// getNextFromNew returns the schedule after an answer to a review card that has
// not been answered. Three of the four ratings schedule it in minutes, and the
// fourth in days.
func (f FSRS) getNextFromNew(one atAnswer, pos time.Time, r fsrs.Rating) Schedule {
	out := one.out
	out.Difficulty = f.first(r)
	out.Stability = math.Max(f.p.W[r-1], 0.1)
	out.Phase = uint8(fsrs.Learning)
	switch r {
	case fsrs.Again:
		out.Due = pos.Add(1 * time.Minute)
	case fsrs.Hard:
		out.Due = pos.Add(5 * time.Minute)
	case fsrs.Good:
		out.Due = pos.Add(10 * time.Minute)
	case fsrs.Easy:
		out.Due = pos.Add(daysToDuration(f.getInterval(out.Stability)))
		out.Phase = uint8(fsrs.Review)
	}
	return out
}

// getNextFromLearning is where an answer leaves a review card the scheduler is
// still putting into memory. The two lower ratings leave it where it was and
// ask again in minutes; the two higher send it away in days and put it into
// review.
func (f FSRS) getNextFromLearning(one atAnswer, pos time.Time, r fsrs.Rating) Schedule {
	out := one.out
	out.Difficulty = f.getNextDifficulty(one.last.Difficulty, r)
	out.Stability = f.getLearningStability(one.last.Stability, r)
	switch r {
	case fsrs.Again:
		out.Due = pos.Add(5 * time.Minute)
	case fsrs.Hard:
		out.Due = pos.Add(10 * time.Minute)
	case fsrs.Good:
		out.Due = pos.Add(daysToDuration(f.getInterval(out.Stability)))
		out.Phase = uint8(fsrs.Review)
	case fsrs.Easy:
		good := f.getInterval(f.getLearningStability(one.last.Stability, fsrs.Good))
		out.Due = pos.Add(daysToDuration(math.Max(f.getInterval(out.Stability), good+1)))
		out.Phase = uint8(fsrs.Review)
	}
	return out
}

// getNextFromReview returns the schedule after an answer to a review card in the
// review phase, computed from its recall probability at the answer instant. The
// Again rating is a lapse and is scheduled again in minutes.
//
// Each interval is at least one day longer than the interval below it, so the due
// day derives from the intervals of the lower ratings and not from the rating's own
// stability alone.
func (f FSRS) getNextFromReview(one atAnswer, pos time.Time, back float64, r fsrs.Rating) Schedule {
	d, s := one.last.Difficulty, one.last.Stability
	out := one.out
	out.Difficulty = f.getNextDifficulty(d, r)
	if r == fsrs.Again {
		out.Stability = math.Min(s/math.Exp(f.p.W[17]*f.p.W[18]), f.getStabilityOnLapse(d, s, back))
		out.Lapses = one.last.Lapses + 1
		out.Phase = uint8(fsrs.Relearning)
		out.Due = pos.Add(5 * time.Minute)
		return out
	}

	out.Stability = f.getStabilityOnRecall(d, s, back, r)
	out.Phase = uint8(fsrs.Review)
	hard := math.Min(
		f.getInterval(f.getStabilityOnRecall(d, s, back, fsrs.Hard)),
		f.getInterval(f.getStabilityOnRecall(d, s, back, fsrs.Good)),
	)
	good := math.Max(f.getInterval(f.getStabilityOnRecall(d, s, back, fsrs.Good)), hard+1)
	switch r {
	case fsrs.Hard:
		out.Due = pos.Add(daysToDuration(hard))
	case fsrs.Easy:
		out.Due = pos.Add(daysToDuration(math.Max(f.getInterval(out.Stability), good+1)))
	case fsrs.Again, fsrs.Good:
		out.Due = pos.Add(daysToDuration(good))
	default:
		out.Due = pos.Add(daysToDuration(good))
	}
	return out
}

// getRecall is how likely a review card was to come back at the instant it was
// answered, on the scheduler's own forgetting curve.
func (f FSRS) getRecall(one atAnswer) float64 {
	return math.Pow(1+f.p.Factor*one.away/one.last.Stability, f.p.Decay)
}

// getInterval returns the number of days an answer schedules a review card of
// this stability, at the scheduler's target retention.
//
// The parameters carry no fuzz, so the interval is exactly the value of this
// arithmetic, and a review card answered the same way receives the same due day
// on every launch.
func (f FSRS) getInterval(stability float64) float64 {
	out := stability / f.p.Factor * (math.Pow(f.p.RequestRetention, 1/f.p.Decay) - 1)
	return math.Max(math.Min(math.Round(out), f.p.MaximumInterval), 1)
}

// first is the difficulty a review card is opened at by the answer that begins it.
func (f FSRS) first(r fsrs.Rating) float64 {
	return clampDifficulty(f.p.W[4] - math.Exp(f.p.W[5]*float64(r-1)) + 1)
}

// getNextDifficulty is where an answer leaves a difficulty, pulled back towards the
// difficulty an easy answer opens a review card at.
func (f FSRS) getNextDifficulty(d float64, r fsrs.Rating) float64 {
	delta := -f.p.W[6] * float64(r-3)
	next := d + (10.0-d)*delta/9.0
	return clampDifficulty(f.p.W[7]*f.first(fsrs.Easy) + (1-f.p.W[7])*next)
}

// getLearningStability is where an answer leaves the stability of a review card
// the scheduler is still putting into memory.
func (f FSRS) getLearningStability(s float64, r fsrs.Rating) float64 {
	return s * math.Exp(f.p.W[17]*(float64(r-3)+f.p.W[18]))
}

// getStabilityOnRecall is where an answer the review card came back on leaves its
// stability. The two ratings either side of a good answer carry a weight of
// their own.
func (f FSRS) getStabilityOnRecall(d, s, back float64, r fsrs.Rating) float64 {
	hard, easy := 1.0, 1.0
	if r == fsrs.Hard {
		hard = f.p.W[15]
	}
	if r == fsrs.Easy {
		easy = f.p.W[16]
	}
	return s * (1 + math.Exp(f.p.W[8])*
		(11-d)*
		math.Pow(s, -f.p.W[9])*
		(math.Exp((1-back)*f.p.W[10])-1)*
		hard*
		easy)
}

// getStabilityOnLapse is where an answer the review card did not come back on
// leaves its stability.
func (f FSRS) getStabilityOnLapse(d, s, back float64) float64 {
	return f.p.W[11] *
		math.Pow(d, -f.p.W[12]) *
		(math.Pow(s+1, f.p.W[13]) - 1) *
		math.Exp((1-back)*f.p.W[14])
}

// clampDifficulty keeps a difficulty inside the scale it is read on.
func clampDifficulty(d float64) float64 { return math.Min(math.Max(d, 1), 10) }

// daysToDuration converts a number of days into a time.Duration.
func daysToDuration(one float64) time.Duration {
	return time.Duration(one) * 24 * time.Hour
}
