package embedding_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// step is what one stored byte is worth: the whole scale over the 127 levels a
// signed byte has on either side of zero.
const step = port.Int8Scale / 127

// widths are the vector widths this application admits: the model it embeds
// with by default is 384 wide, and the settings take any model a person names,
// which reaches 3072 among the ones a service serves.
var widths = []int{384, 768, 1024, 1536, 3072}

// unit is an invented vector of the width given, pointing nowhere in
// particular. Its length is one, so its components are around 1/sqrt(d) and the
// number of levels a byte spends on it falls as it grows wider.
func unit(rng *rand.Rand, d int) []float32 {
	v := make([]float32, d)
	for i := range v {
		v[i] = float32(rng.NormFloat64())
	}
	return embedding.Normalise(v)
}

// exact is the cosine of two vectors before either is quantised, which is what
// the rerank would order by if it could afford to keep floats.
func exact(a, b []float32) float64 {
	var dot, left, right float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		left += float64(a[i]) * float64(a[i])
		right += float64(b[i]) * float64(b[i])
	}
	return dot / math.Sqrt(left*right)
}

// TestOneScaleServesEveryWidth is why the scale is one number and not a number
// per model.
//
// A wider vector spends fewer of the 127 levels: its components are around
// 1/sqrt(d), so at 3072 dimensions the typical byte is single figures. That
// looks like a scale fitted to one width and wasted on the others, and it is
// not. What the rerank orders by is a cosine, and the error quantising leaves
// in a cosine is the error left in one component: the query's own components
// shrink as 1/sqrt(d) while d of them are summed, and the two cancel. Levels
// spent falls with the width; the answer does not move.
func TestOneScaleServesEveryWidth(t *testing.T) {
	rng := rand.New(rand.NewPCG(20260903, 7))
	for _, d := range widths {
		var levels, mean, worst float64
		const trials = 200
		for range trials {
			query, doc := unit(rng, d), unit(rng, d)
			stored := embedding.Bytes(doc)
			for _, q := range stored {
				levels += math.Abs(float64(q))
			}
			off := math.Abs(exact(query, doc) - embedding.Similarity(query, stored))
			mean += off
			worst = math.Max(worst, off)
		}
		levels /= trials * float64(d)
		mean /= trials

		t.Logf("%4d dimensions: %5.1f of 127 levels, similarity off by %.2e on average, %.2e at worst",
			d, levels, mean, worst)

		if mean > step/2 {
			t.Errorf("%d dimensions: the similarity is off by %.2e on average, more than half a step of %.2e",
				d, mean, step)
		}
		if worst > 3*step {
			t.Errorf("%d dimensions: the similarity is off by %.2e at worst, more than three steps of %.2e",
				d, worst, step)
		}
	}
}

// TestQuantisingCostsTheSameAtEveryWidth is the same statement without the
// table: whatever width a model answers at, a stored vector puts the rerank's
// similarity within a few of the steps the scale sets.
//
// The width is drawn from 256 upwards because below that a unit-length vector
// starts to carry a component beyond the scale, which is clamped, and clamping
// is what the scale is chosen to leave room for rather than what it bounds.
func TestQuantisingCostsTheSameAtEveryWidth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := rapid.IntRange(256, 4096).Draw(t, "dimensions")
		rng := rand.New(rand.NewPCG(rapid.Uint64().Draw(t, "seed"), 1))

		query, doc := unit(rng, d), unit(rng, d)
		stored := embedding.Bytes(doc)
		for i, q := range stored {
			if q == 127 || q == -127 {
				t.Fatalf("dimension %d of a unit vector of %d reached the scale", i, d)
			}
		}
		if off := math.Abs(exact(query, doc) - embedding.Similarity(query, stored)); off > 4*step {
			t.Fatalf("%d dimensions: the similarity is off by %.2e, more than four steps of %.2e",
				d, off, step)
		}
	})
}
