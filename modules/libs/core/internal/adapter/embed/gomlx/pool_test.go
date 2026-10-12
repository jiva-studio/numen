package gomlx

import (
	"math"
	"slices"
	"testing"
)

func TestPaddingIsNotAveragedIn(t *testing.T) {
	// Two texts of one token in a batch padded to three. The padded positions
	// hold a value the mask excludes.
	flat := []float32{
		3, 0, 100, 100, 100, 100,
		0, 3, 100, 100, 100, 100,
	}
	mask := []int64{1, 0, 0, 1, 0, 0}
	got := meanPool(flat, mask, 2, 3, 2)
	if !slices.Equal(got[0], []float32{1, 0}) || !slices.Equal(got[1], []float32{0, 1}) {
		t.Errorf("got %v", got)
	}
}

func TestTokensAreAveragedAndTheResultIsUnitLength(t *testing.T) {
	flat := []float32{
		1, 0,
		0, 1,
	}
	got := meanPool(flat, []int64{1, 1}, 1, 2, 2)
	want := float32(math.Sqrt2 / 2)
	if math.Abs(float64(got[0][0]-want)) > 1e-6 || math.Abs(float64(got[0][1]-want)) > 1e-6 {
		t.Errorf("got %v, want %v", got[0], want)
	}
}

func TestTheFirstTokenIsTheVectorWhenTheModelPoolsThatWay(t *testing.T) {
	// Two texts of three tokens. Only the first token of each carries the
	// vector; a model trained this way puts nothing in the rest.
	flat := []float32{
		3, 0, 9, 9, 9, 9,
		0, 5, 9, 9, 9, 9,
	}
	got := headPool(flat, 2, 3, 2)
	if !slices.Equal(got[0], []float32{1, 0}) || !slices.Equal(got[1], []float32{0, 1}) {
		t.Errorf("got %v", got)
	}
}

func TestAllPaddingIsAZeroVector(t *testing.T) {
	got := meanPool([]float32{5, 5}, []int64{0}, 1, 1, 2)
	if !slices.Equal(got[0], []float32{0, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestSequenceLengthsRoundUpToAStep(t *testing.T) {
	for _, c := range []struct {
		tokens, want int
	}{
		{1, 64},
		{64, 64},
		{65, 128},
		{200, 256},
		{256, 256},
		{9000, 256},
	} {
		if got := bucket(c.tokens, 64, 256); got != c.want {
			t.Errorf("%d tokens gave %d, want %d", c.tokens, got, c.want)
		}
	}
}

// A shape is compiled the first time it appears, so a batch that is not full is
// laid out in as many rows as a full one and padding fills the rest.
func TestABatchIsLaidOutAtTheSizeAFullOneCarries(t *testing.T) {
	const rows, seq = 8, 64
	for texts := 1; texts <= rows; texts++ {
		batch := make([][]int, texts)
		for i := range batch {
			batch[i] = []int{7, 8, 9}
		}
		in := padBatch(batch, rows, seq, 1)
		ids, mask, types := in.ids, in.mask, in.types
		for _, held := range [][]int64{ids, mask, types} {
			if len(held) != rows*seq {
				t.Fatalf("%d texts were laid out in %d elements", texts, len(held))
			}
		}
		// A row nothing was written into is padding, and what pools it divides
		// by the tokens it is marked at.
		for row := texts; row < rows; row++ {
			at := row * seq
			if ids[at] != 1 {
				t.Errorf("row %d of %d holds %d", row, texts, ids[at])
			}
			var marked int64
			for _, v := range mask[at : at+seq] {
				marked += v
			}
			if marked != 1 {
				t.Errorf("row %d of %d is marked at %d tokens", row, texts, marked)
			}
		}
		releasePadding(in)
	}
}

// The shapes one run meets are the sequence lengths and no more, whatever a
// batch holds.
func TestARunMeetsOneShapePerSequenceLength(t *testing.T) {
	const rows, limit, step = 8, 256, 64
	seen := map[[2]int]bool{}
	for texts := 1; texts <= rows; texts++ {
		for _, tokens := range []int{1, 40, 65, 200, 300} {
			batch := make([][]int, texts)
			for i := range batch {
				batch[i] = make([]int, tokens)
			}
			in := padBatch(batch, rows, bucket(tokens, step, limit), 1)
			seen[[2]int{rows, len(in.ids) / rows}] = true
			releasePadding(in)
		}
	}
	if want := limit/step + 2; len(seen) > want {
		t.Errorf("a run met %d shapes, and the cache holds %d", len(seen), want)
	}
}

func BenchmarkMeanPool(b *testing.B) {
	const rows, seq, dimensions = 8, 512, 1024
	flat := make([]float32, rows*seq*dimensions)
	for i := range flat {
		flat[i] = float32(i%97) * 0.01
	}
	mask := make([]int64, rows*seq)
	for row := range rows {
		for token := 0; token < seq-row*16; token++ {
			mask[row*seq+token] = 1
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		meanPool(flat, mask, rows, seq, dimensions)
	}
}

func BenchmarkPadBatch(b *testing.B) {
	batch := make([][]int, 32)
	for row := range batch {
		batch[row] = make([]int, 512-row*8)
		for i := range batch[row] {
			batch[row][i] = i + 1
		}
	}
	const rows, seq = 32, 512
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		in := padBatch(batch, rows, seq, 0)
		releasePadding(in)
	}
}

// Each text's vector is its own first token, wherever the text stands in the batch.
func TestEachTextTakesItsOwnFirstTokenWhateverItsPlace(t *testing.T) {
	const rows, seq, dimensions = 3, 4, 3
	flat := make([]float32, rows*seq*dimensions)
	for i := range flat {
		flat[i] = 9
	}
	for row := range rows {
		at := row * seq * dimensions
		copy(flat[at:at+dimensions], []float32{0, 0, 0})
		flat[at+row] = 2
	}
	got := headPool(flat, rows, seq, dimensions)
	for row := range rows {
		want := make([]float32, dimensions)
		want[row] = 1
		if !slices.Equal(got[row], want) {
			t.Errorf("row %d: got %v, want %v", row, got[row], want)
		}
	}
}

func TestAnEmptyTextIsHeldToOneStep(t *testing.T) {
	for _, tokens := range []int{0, -3} {
		if got := bucket(tokens, 64, 256); got != 64 {
			t.Errorf("%d tokens gave %d, want 64", tokens, got)
		}
	}
}

func TestALimitThatIsNoStepIsNeverPassed(t *testing.T) {
	for _, c := range []struct {
		tokens, want int
	}{
		{150, 192},
		{193, 200},
		{200, 200},
		{500, 200},
	} {
		if got := bucket(c.tokens, 64, 200); got != c.want {
			t.Errorf("%d tokens gave %d, want %d", c.tokens, got, c.want)
		}
	}
}

// A padding handed back is the one the next batch of its size is laid out on,
// slices and all.
func TestAReleasedPaddingIsReusedForTheNextBatchOfItsSize(t *testing.T) {
	// Larger than any padding another batch left in the pool, so the first one is made to this size exactly.
	batch := [][]int{{1, 2, 3}, {4, 5}}
	reused := 0
	previous := padBatch(batch, 2, 1<<15, 0)
	for range 100 {
		ids, mask, types := &previous.ids[0], &previous.mask[0], &previous.types[0]
		releasePadding(previous)
		next := padBatch(batch, 2, 1<<15, 0)
		if next == previous {
			reused++
			if &next.ids[0] != ids || &next.mask[0] != mask || &next.types[0] != types {
				t.Fatal("a reused padding was laid out on new slices")
			}
		}
		previous = next
	}
	if reused == 0 {
		t.Error("no padding handed back was ever reused")
	}
}
