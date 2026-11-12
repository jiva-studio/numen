package vector_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vector"
)

// step is what one stored byte is worth: the whole scale over the 127 levels a
// signed byte has on either side of zero.
const step = vector.Int8Scale / 127

func getWidths() []int { return []int{384, 768, 1024, 1536, 3072} }

func unit(rng *rand.Rand, d int) []float32 {
	v := make([]float32, d)
	for i := range v {
		v[i] = float32(rng.NormFloat64())
	}
	return vector.Normalize(v)
}

func exact(a, b []float32) float64 {
	var dot, remaining, right float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		remaining += float64(a[i]) * float64(a[i])
		right += float64(b[i]) * float64(b[i])
	}
	return dot / math.Sqrt(remaining*right)
}

// TestOneScaleServesEveryWidth is why the scale is one number and not one per model.
// A wider vector spends fewer of the 127 levels (components are around
// 1/sqrt(d)), but the rerank orders by a cosine: the query's components shrink
// as 1/sqrt(d) while d of them are summed, and the two cancel, so the
// quantization error in the cosine does not move with the width.
func TestOneScaleServesEveryWidth(t *testing.T) {
	rng := rand.New(rand.NewPCG(20260903, 7))
	for _, d := range getWidths() {
		var levels, mean, worst float64
		const trials = 200
		for range trials {
			query, doc := unit(rng, d), unit(rng, d)
			stored := vector.Bytes(doc)
			for _, q := range stored {
				levels += math.Abs(float64(q))
			}
			off := math.Abs(exact(query, doc) - similarity(query, stored))
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

// TestQuantizingCostsTheSameAtEveryWidth checks that at any width a stored
// vector puts the rerank's similarity within a few steps of the scale.
// The width is drawn from 256 upwards: below that a unit-length vector starts
// to carry a component beyond the scale, which is clamped, and the scale has
// headroom for that clamping.
func TestQuantizingCostsTheSameAtEveryWidth(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := rapid.IntRange(256, 4096).Draw(t, "dimensions")
		rng := rand.New(rand.NewPCG(rapid.Uint64().Draw(t, "seed"), 1))

		query, doc := unit(rng, d), unit(rng, d)
		stored := vector.Bytes(doc)
		for i, q := range stored {
			if q == 127 || q == -127 {
				t.Fatalf("dimension %d of a unit vector of %d reached the scale", i, d)
			}
		}
		if off := math.Abs(exact(query, doc) - similarity(query, stored)); off > 4*step {
			t.Fatalf("%d dimensions: the similarity is off by %.2e, more than four steps of %.2e",
				d, off, step)
		}
	})
}

func similarity(query []float32, stored []int8) float64 {
	raw := make([]byte, len(stored))
	for i, q := range stored {
		raw[i] = byte(q)
	}
	return vector.SimilarityBytes(query, raw)
}
