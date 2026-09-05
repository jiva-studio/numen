package embedding_test

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/embedding"
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

func TestBytesRoundTrip(t *testing.T) {
	step := embedding.Int8Scale / 127
	for _, c := range []struct {
		name string
		in   float32
		want int8
	}{
		{"zero", 0, 0},
		{"one step", float32(step), 1},
		{"half the scale", embedding.Int8Scale / 2, 64},
		{"the scale itself", embedding.Int8Scale, 127},
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
			back := float64(got[0]) / 127 * embedding.Int8Scale
			want := math.Min(math.Abs(float64(c.in)), embedding.Int8Scale)
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
			got := embedding.Similarity(c.query, c.stored)
			if math.Abs(got-c.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, c.want)
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

	got := embedding.Similarity(v, embedding.Bytes(v))
	if got < 0.999 {
		t.Errorf("a vector against its own quantisation stands at %v", got)
	}
}
