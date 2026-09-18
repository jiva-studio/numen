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
// The parameters are the library's and the arithmetic over them is here, so a
// scheduler carries no state between answers and one may be asked about many
// card faces at once.
type FSRS struct {
	p    fsrs.Parameters
	name string
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
	return FSRS{p: p, name: FSRSName + "." + hashParameters(p)}
}

// NewFSRSAt is the scheduler asking for this share of the cards to come back
// when they come round. A share outside what a preset may hold is brought to
// the nearest end of it.
func NewFSRSAt(retention float64) FSRS {
	p := fsrs.DefaultParam()
	p.EnableFuzz = false
	p.RequestRetention = math.Min(math.Max(retention, RetentionBounds.Least), RetentionBounds.Most)
	return FSRS{p: p, name: FSRSName + "." + hashParameters(p)}
}

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

// IsSpaced reports whether this scheduler has put a card face into review: it
// has been answered well enough to come round in days.
func (FSRS) IsSpaced(s Schedule) bool {
	return s.IsSeen() && fsrs.State(s.Phase) == fsrs.Review
}

// Next is where an answer leaves a schedule.
func (f FSRS) Next(s Schedule, at time.Time, r Rating) Schedule {
	one := f.readAtAnswer(s, at)
	switch one.phase {
	case fsrs.New:
		return f.getNextFromNew(one, at, fsrs.Rating(r))
	case fsrs.Review:
		return f.getNextFromReview(one, at, f.getRecall(one), fsrs.Rating(r))
	default:
		return f.getNextFromLearning(one, at, fsrs.Rating(r))
	}
}

// Endings is where the two endings leave a card face.
//
// A card face this scheduler has put into review is settled at every rating by
// one reckoning of how likely it was to come back, so both endings are worked
// out from that one reckoning.
func (f FSRS) Endings(s Schedule, at time.Time) (good, again Schedule) {
	one := f.readAtAnswer(s, at)
	switch one.phase {
	case fsrs.New:
		return f.getNextFromNew(one, at, fsrs.Good), f.getNextFromNew(one, at, fsrs.Again)
	case fsrs.Review:
		back := f.getRecall(one)
		return f.getNextFromReview(one, at, back, fsrs.Good),
			f.getNextFromReview(one, at, back, fsrs.Again)
	default:
		return f.getNextFromLearning(one, at, fsrs.Good),
			f.getNextFromLearning(one, at, fsrs.Again)
	}
}

// atAnswer is a card face at the instant it is answered: the phase the answer
// finds it in, how many whole days it stood away, and where the answer leaves
// what no rating decides.
type atAnswer struct {
	phase fsrs.State
	// away is how many whole days the card face stood away, and is none for one
	// nobody has answered.
	away float64
	// last is where the card face stood before the answer, and out is where the
	// answer leaves it but for what the rating decides.
	last Schedule
	out  Schedule
}

// readAtAnswer reads a card face at the instant it is answered. One nobody has
// answered opens at the phase the scheduler begins a card in, at no stability
// and no difficulty.
func (f FSRS) readAtAnswer(s Schedule, at time.Time) atAnswer {
	one := atAnswer{phase: fsrs.New}
	if s.IsSeen() {
		one.last, one.phase = s, fsrs.State(s.Phase)
	}
	// The days away are the whole days gone by, and they do not go below none.
	if one.phase != fsrs.New {
		one.away = math.Max(math.Floor(at.Sub(one.last.Last).Hours()/24), 0)
	}
	one.out = one.last
	one.out.Last = at
	one.out.Reps = one.last.Reps + 1
	return one
}

// getNextFromNew is where an answer leaves a card face nobody had answered.
// Three of the four put it into memory over minutes, and the fourth sends it
// away in days.
func (f FSRS) getNextFromNew(one atAnswer, at time.Time, r fsrs.Rating) Schedule {
	out := one.out
	out.Difficulty = f.first(r)
	out.Stability = math.Max(f.p.W[r-1], 0.1)
	out.Phase = uint8(fsrs.Learning)
	switch r {
	case fsrs.Again:
		out.Due = at.Add(1 * time.Minute)
	case fsrs.Hard:
		out.Due = at.Add(5 * time.Minute)
	case fsrs.Good:
		out.Due = at.Add(10 * time.Minute)
	case fsrs.Easy:
		out.Due = at.Add(days(f.getInterval(out.Stability)))
		out.Phase = uint8(fsrs.Review)
	}
	return out
}

// getNextFromLearning is where an answer leaves a card face the scheduler is
// still putting into memory. The two lower ratings leave it where it was and
// ask again in minutes; the two higher send it away in days and put it into
// review.
func (f FSRS) getNextFromLearning(one atAnswer, at time.Time, r fsrs.Rating) Schedule {
	out := one.out
	out.Difficulty = f.harder(one.last.Difficulty, r)
	out.Stability = f.getLearningStability(one.last.Stability, r)
	switch r {
	case fsrs.Again:
		out.Due = at.Add(5 * time.Minute)
	case fsrs.Hard:
		out.Due = at.Add(10 * time.Minute)
	case fsrs.Good:
		out.Due = at.Add(days(f.getInterval(out.Stability)))
		out.Phase = uint8(fsrs.Review)
	case fsrs.Easy:
		good := f.getInterval(f.getLearningStability(one.last.Stability, fsrs.Good))
		out.Due = at.Add(days(math.Max(f.getInterval(out.Stability), good+1)))
		out.Phase = uint8(fsrs.Review)
	}
	return out
}

// getNextFromReview is where an answer leaves a card face the scheduler has put
// into review, worked out from how likely it was to come back. The ending it
// did not come back on is a lapse and is asked again in minutes.
//
// Each interval is held past the one below it, so the day an answer names is
// worked out from the endings under it and not from its own stability alone.
func (f FSRS) getNextFromReview(one atAnswer, at time.Time, back float64, r fsrs.Rating) Schedule {
	d, s := one.last.Difficulty, one.last.Stability
	out := one.out
	out.Difficulty = f.harder(d, r)
	if r == fsrs.Again {
		out.Stability = math.Min(s/math.Exp(f.p.W[17]*f.p.W[18]), f.getStabilityOnLapse(d, s, back))
		out.Lapses = one.last.Lapses + 1
		out.Phase = uint8(fsrs.Relearning)
		out.Due = at.Add(5 * time.Minute)
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
		out.Due = at.Add(days(hard))
	case fsrs.Easy:
		out.Due = at.Add(days(math.Max(f.getInterval(out.Stability), good+1)))
	default:
		out.Due = at.Add(days(good))
	}
	return out
}

// getRecall is how likely a card face was to come back at the instant it was
// answered, on the scheduler's own forgetting curve.
func (f FSRS) getRecall(one atAnswer) float64 {
	return math.Pow(1+f.p.Factor*one.away/one.last.Stability, f.p.Decay)
}

// getInterval is how many days an answer sends a card face standing at this
// stability away for, at the share of the cards this scheduler asks to bring
// back.
//
// The parameters carry no fuzz, so the interval is the number this arithmetic
// gives, and a card face answered the same way is sent away for the same day at
// every launch.
func (f FSRS) getInterval(stability float64) float64 {
	out := stability / f.p.Factor * (math.Pow(f.p.RequestRetention, 1/f.p.Decay) - 1)
	return math.Max(math.Min(math.Round(out), f.p.MaximumInterval), 1)
}

// first is the difficulty a card face is opened at by the answer that begins it.
func (f FSRS) first(r fsrs.Rating) float64 {
	return clampDifficulty(f.p.W[4] - math.Exp(f.p.W[5]*float64(r-1)) + 1)
}

// harder is where an answer leaves a difficulty, pulled back towards the
// difficulty an easy answer opens a card face at.
func (f FSRS) harder(d float64, r fsrs.Rating) float64 {
	delta := -f.p.W[6] * float64(r-3)
	next := d + (10.0-d)*delta/9.0
	return clampDifficulty(f.p.W[7]*f.first(fsrs.Easy) + (1-f.p.W[7])*next)
}

// getLearningStability is where an answer leaves the stability of a card face
// the scheduler is still putting into memory.
func (f FSRS) getLearningStability(s float64, r fsrs.Rating) float64 {
	return s * math.Exp(f.p.W[17]*(float64(r-3)+f.p.W[18]))
}

// getStabilityOnRecall is where an answer the card face came back on leaves its
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

// getStabilityOnLapse is where an answer the card face did not come back on
// leaves its stability.
func (f FSRS) getStabilityOnLapse(d, s, back float64) float64 {
	return f.p.W[11] *
		math.Pow(d, -f.p.W[12]) *
		(math.Pow(s+1, f.p.W[13]) - 1) *
		math.Exp((1-back)*f.p.W[14])
}

// clampDifficulty keeps a difficulty inside the scale it is read on.
func clampDifficulty(d float64) float64 { return math.Min(math.Max(d, 1), 10) }

// days is a whole number of days as a length of time.
func days(one float64) time.Duration {
	return time.Duration(one) * 24 * time.Hour
}
