package onnx

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

func TestSequenceBucketing(t *testing.T) {
	for _, c := range []struct {
		length int
		want   int
	}{
		{0, 64},
		{1, 64},
		{63, 64},
		{64, 64},
		{65, 128},
		{128, 128},
		{129, 256},
		{256, 256},
		{257, 512},
		{512, 512},
		{600, 512},
	} {
		tokens := make([]int, c.length)
		_, seq, in := padBatch([][]int{tokens}, 0)
		releasePadding(in)
		if seq != c.want {
			t.Errorf("length %d was bucketed to %d, want %d", c.length, seq, c.want)
		}
	}
}

// A batch is laid out at the discrete bucket boundary of its longest text.
func TestABatchIsPaddedToBucketBoundary(t *testing.T) {
	rows, seq, in := padBatch([][]int{{7, 8, 9}, {4}}, 1)
	defer releasePadding(in)
	ids, mask, types := in.ids, in.mask, in.types
	if rows != 2 || seq != 64 {
		t.Fatalf("two texts of three and one tokens were laid out %dx%d, want 2x64", rows, seq)
	}
	for _, held := range [][]int64{ids, mask, types} {
		if len(held) != rows*seq {
			t.Fatalf("a row of %d holds %d", seq, len(held))
		}
	}
	if !slices.Equal(ids[:3], []int64{7, 8, 9}) || ids[3] != 1 || ids[63] != 1 {
		t.Errorf("the first text was padded incorrectly: %v", ids[:64])
	}
	if ids[64] != 4 || ids[65] != 1 || ids[127] != 1 {
		t.Errorf("the second text was padded incorrectly: %v", ids[64:])
	}
	if !slices.Equal(mask[:4], []int64{1, 1, 1, 0}) || mask[63] != 0 {
		t.Errorf("the first mask is marked: %v", mask[:64])
	}
	if !slices.Equal(mask[64:66], []int64{1, 0}) || mask[127] != 0 {
		t.Errorf("the second mask is marked: %v", mask[64:])
	}
}

// A text the tokenizer gave nothing for is still a text, and what pools its row
// divides by something.
func TestAnEmptyTextIsMarkedAtOneToken(t *testing.T) {
	rows, seq, in := padBatch([][]int{{}}, 1)
	defer releasePadding(in)
	mask := in.mask
	if rows != 1 || seq != 64 {
		t.Fatalf("one empty text was laid out %dx%d, want 1x64", rows, seq)
	}
	if mask[0] != 1 || slices.Contains(mask[1:], 1) {
		t.Errorf("an empty text is marked %v", mask)
	}
}

func TestAnOutputIsChosenByName(t *testing.T) {
	for _, c := range []struct {
		named []string
		want  string
	}{
		{[]string{"last_hidden_state", "pooler_output"}, "last_hidden_state"},
		{[]string{"pooler_output", "sentence_embedding"}, "sentence_embedding"},
		{[]string{"token_embeddings", "sentence_embedding"}, "sentence_embedding"},
		{[]string{"the_only_one"}, "the_only_one"},
	} {
		got, err := chooseOutput(c.named)
		if err != nil {
			t.Errorf("%v: %v", c.named, err)
			continue
		}
		if got != c.want {
			t.Errorf("%v gave %q, want %q", c.named, got, c.want)
		}
	}
	if _, err := chooseOutput([]string{"one", "another"}); err == nil {
		t.Error("a model naming neither and answering twice was taken at a guess")
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
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _, in := padBatch(batch, 0)
		releasePadding(in)
	}
}
