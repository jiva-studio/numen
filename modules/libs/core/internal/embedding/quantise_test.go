package embedding_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

func TestBitsKeepTheSign(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []float32
		want []byte
	}{
		{"one positive dimension", []float32{0.5}, []byte{0b1000_0000}},
		{"one negative dimension", []float32{-0.5}, []byte{0}},
		{"zero is not positive", []float32{0}, []byte{0}},
		{"first and last of a byte", []float32{1, -1, -1, -1, -1, -1, -1, 1}, []byte{0b1000_0001}},
		{"a ninth dimension starts a second byte", []float32{-1, -1, -1, -1, -1, -1, -1, -1, 1}, []byte{0, 0b1000_0000}},
		{"a partial byte is padded with zeros", []float32{1, 1, 1}, []byte{0b1110_0000}},
		{"no dimensions", []float32{}, []byte{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := embedding.Bits(c.in); !slices.Equal(got, c.want) {
				t.Errorf("got %08b, want %08b", got, c.want)
			}
		})
	}
}

func TestCoarseKeepsTheSignOfAQuantisedVector(t *testing.T) {
	for _, c := range []struct {
		name string
		in   []int8
		want []byte
	}{
		{"one positive dimension", []int8{1}, []byte{0b1000_0000}},
		{"one negative dimension", []int8{-1}, []byte{0}},
		{"zero is not positive", []int8{0}, []byte{0}},
		{"first and last of a byte", []int8{127, -1, -1, -1, -1, -1, -1, 5}, []byte{0b1000_0001}},
		{"a ninth dimension starts a second byte", []int8{-1, -1, -1, -1, -1, -1, -1, -1, 1}, []byte{0, 0b1000_0000}},
		{"a partial byte is padded with zeros", []int8{1, 1, 1}, []byte{0b1110_0000}},
		{"every dimension of two bytes", []int8{1, -1, 1, 1, -1, 1, -1, -1, 1, 1, 1, -1, -1, -1, 1, 1}, []byte{0b1011_0100, 0b1110_0011}},
		{"no dimensions", []int8{}, []byte{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := embedding.Coarse(c.in); got == nil || !slices.Equal(got, c.want) {
				t.Errorf("got %#v, want %08b", got, c.want)
			}
		})
	}
}

func TestBytesRoundTrip(t *testing.T) {
	step := port.Int8Scale / 127
	for _, c := range []struct {
		name string
		in   float32
		want int8
	}{
		{"zero", 0, 0},
		{"one step", float32(step), 1},
		{"half the scale", port.Int8Scale / 2, 64},
		{"the scale itself", port.Int8Scale, 127},
		{"beyond the scale is clamped", 1, 127},
		{"beyond the scale, negative", -1, -127},
		{"a value between two steps rounds", float32(step * 2.7), 3},
		{"negative rounds the same way", float32(-step * 2.7), -3},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := embedding.Bytes([]float32{c.in})
			if got[0] != c.want {
				t.Fatalf("got %d, want %d", got[0], c.want)
			}
			back := float64(got[0]) / 127 * port.Int8Scale
			want := math.Min(math.Abs(float64(c.in)), port.Int8Scale)
			if diff := math.Abs(math.Abs(back) - want); diff > step/2 {
				t.Errorf("read back %v from %v, off by %v", back, c.in, diff)
			}
		})
	}
}

func TestDimensionsReadEveryStoredByteBack(t *testing.T) {
	stored := make([]byte, 256)
	want := make([]int8, 256)
	for i := range stored {
		stored[i] = byte(i)
		want[i] = int8(i)
	}
	if got := embedding.Dimensions(stored); !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := embedding.Dimensions(nil); len(got) != 0 {
		t.Errorf("got %v for no vector", got)
	}
}

// The same vector quantises to the same bytes in a run over any other data.
// This is the property that keeps a vector stored today comparable with one
// stored after a book in another language is added.
func TestQuantisationDoesNotDependOnTheRestOfTheCorpus(t *testing.T) {
	vector := embedding.Normalise([]float32{0.1, -0.4, 0.02, 0.9, -0.01, 0.3, 0.0, -0.7})

	first := embedding.Bytes(vector)
	firstBits := embedding.Bits(vector)

	// A second run over a corpus with a different spread — every other vector
	// far larger, as a book of another language would be.
	crowd := rand.New(rand.NewPCG(7, 11))
	for range 100 {
		other := make([]float32, len(vector))
		for i := range other {
			other[i] = float32(crowd.Float64()*40 - 20)
		}
		embedding.Bytes(other)
		embedding.Bits(other)
	}
	second := embedding.Bytes(vector)
	secondBits := embedding.Bits(vector)

	if !slices.Equal(first, second) {
		t.Errorf("bytes moved: %v then %v", first, second)
	}
	if !slices.Equal(firstBits, secondBits) {
		t.Errorf("bits moved: %08b then %08b", firstBits, secondBits)
	}
}

func TestNormaliseGivesUnitLength(t *testing.T) {
	got := embedding.Normalise([]float32{3, 4})
	if !slices.Equal(got, []float32{0.6, 0.8}) {
		t.Errorf("got %v", got)
	}
}

func TestNormaliseKeepsAllZeros(t *testing.T) {
	if got := embedding.Normalise([]float32{0, 0}); !slices.Equal(got, []float32{0, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestSimilarityIsTheAngleBetweenTwoVectors(t *testing.T) {
	for _, c := range []struct {
		name   string
		query  []float32
		stored []int8
		want   float64
	}{
		{"the same direction", []float32{1, 1, 1, 1}, []int8{7, 7, 7, 7}, 1},
		{"the opposite direction", []float32{1, 1, 1, 1}, []int8{-7, -7, -7, -7}, -1},
		{"at a right angle", []float32{1, 1, 0, 0}, []int8{0, 0, 7, 7}, 0},
		{"an eighth of the dimensions turned", []float32{1, 1, 1, 1, 1, 1, 1, -1}, []int8{1, 1, 1, 1, 1, 1, 1, 1}, 0.75},
		{"length does not enter", []float32{9, 0}, []int8{1, 0}, 1},
		{"widths that do not match", []float32{1, 1}, []int8{1}, 0},
		{"a vector with no direction", []float32{1, 1}, []int8{0, 0}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			bytes := make([]byte, len(c.stored))
			for i, v := range c.stored {
				bytes[i] = byte(v)
			}
			gotBytes := embedding.SimilarityBytes(c.query, bytes)
			if math.Abs(gotBytes-c.want) > 1e-9 {
				t.Errorf("gotBytes %v, want %v", gotBytes, c.want)
			}
		})
	}
}

func TestSimilarityIsOneForAVectorWithItself(t *testing.T) {
	// A stored vector is the query's own, quantised. The angle between them is
	// what the quantisation cost, and it is small enough for a floor to be set
	// in the same units either side of it.
	random := rand.New(rand.NewPCG(7, 11))
	v := make([]float32, 1024)
	for i := range v {
		v[i] = random.Float32()*2 - 1
	}
	embedding.Normalise(v)

	quantised := embedding.Bytes(v)
	raw := make([]byte, len(quantised))
	for i, q := range quantised {
		raw[i] = byte(q)
	}
	got := embedding.SimilarityBytes(v, raw)
	if got < 0.999 {
		t.Errorf("a vector against its own quantisation stands at %v", got)
	}
}

func BenchmarkDimensions(b *testing.B) {
	for _, dim := range []int{384, 768, 1024, 1536, 3072} {
		b.Run(strconv.Itoa(dim), func(b *testing.B) {
			raw := make([]byte, dim)
			for i := range raw {
				raw[i] = byte(i)
			}
			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				_ = embedding.Dimensions(raw)
			}
		})
	}
}

func BenchmarkAsDimensions(b *testing.B) {
	BenchmarkDimensions(b)
}

func TestBytesClampAtTheLastLevelEitherSide(t *testing.T) {
	for _, c := range []struct {
		name string
		in   float32
		want int8
	}{
		{"a hundred steps", float32(step * 100), 100},
		{"a hundred steps, negative", float32(-step * 100), -100},
		{"one step past the last level", float32(step * 128), 127},
		{"one step past the last level, negative", float32(-step * 128), -127},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := embedding.Bytes([]float32{c.in}); got[0] != c.want {
				t.Errorf("got %d, want %d", got[0], c.want)
			}
		})
	}
}

func TestDimensionsOfAnEmptyVectorAreNone(t *testing.T) {
	if got := embedding.Dimensions([]byte{}); got != nil {
		t.Errorf("got %#v for an empty stored vector, want nil", got)
	}
	if got := embedding.Dimensions([]byte{0xff}); !slices.Equal(got, []int8{-1}) {
		t.Errorf("got %v for one stored byte, want [-1]", got)
	}
}

// The similarity is the cosine, at every width: those the
// unrolled loop covers whole and those with dimensions left over.
func TestSimilarityIsTheCosineAtEveryWidth(t *testing.T) {
	random := rand.New(rand.NewPCG(3, 5))
	for width := 1; width <= 13; width++ {
		query := make([]float32, width)
		stored := make([]int8, width)
		raw := make([]byte, width)
		for i := range width {
			query[i] = random.Float32()*2 - 1
			stored[i] = int8(random.IntN(254) - 127)
			if stored[i] >= 0 {
				stored[i]++
			}
			raw[i] = byte(stored[i])
		}
		want := getCosine(query, stored)
		if got := embedding.SimilarityBytes(query, raw); !(math.Abs(got-want) <= 1e-5) {
			t.Errorf("width %d: SimilarityBytes = %v, want %v", width, got, want)
		}
	}
}

// A side with no direction answers exactly zero, whatever the width.
func TestSimilarityWithNoDirectionIsZero(t *testing.T) {
	for _, c := range []struct {
		name   string
		query  []float32
		stored []int8
		want   float64
	}{
		{"a query of zeros", []float32{0, 0, 0, 0, 0}, []int8{1, 2, 3, 4, 5}, 0},
		{"a stored vector of zeros", []float32{1, 2, 3, 4, 5}, []int8{0, 0, 0, 0, 0}, 0},
		{"both of zeros", []float32{0}, []int8{0}, 0},
		{"one dimension each way", []float32{1}, []int8{5}, 1},
		{"one dimension against zero", []float32{1}, []int8{0}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := make([]byte, len(c.stored))
			for i, v := range c.stored {
				raw[i] = byte(v)
			}
			if got := embedding.SimilarityBytes(c.query, raw); got != c.want {
				t.Errorf("SimilarityBytes = %v, want %v", got, c.want)
			}
		})
	}
}

// getCosine is the cosine of the angle between two vectors, summed in float64.
func getCosine(query []float32, stored []int8) float64 {
	var dot, remaining, right float64
	for i := range query {
		q, s := float64(query[i]), float64(stored[i])
		dot += q * s
		remaining += q * q
		right += s * s
	}
	return dot / math.Sqrt(remaining*right)
}

// Four elements are summed a step in single precision, so a sum of 2^24 and 1
// stays at 2^24.
func TestSimilaritySumsFourElementsAStepInSinglePrecision(t *testing.T) {
	query := []float32{1 << 24, 1, 0, 0}
	want := math.Sqrt2 / 2
	if got := embedding.SimilarityBytes(query, []byte{1, 1, 0, 0}); math.Abs(got-want) > 1e-12 {
		t.Errorf("SimilarityBytes = %v, want %v", got, want)
	}
}
