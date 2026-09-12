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

// A batch carries what its texts hold and no more: a short batch is not laid out
// at the length of a long one.
func TestABatchIsAsLongAsItsLongestText(t *testing.T) {
	rows, seq, ids, mask, types := padBatch([][]int{{7, 8, 9}, {4}}, 1)
	if rows != 2 || seq != 3 {
		t.Fatalf("two texts of three and one tokens were laid out %dx%d", rows, seq)
	}
	for _, held := range [][]int64{ids, mask, types} {
		if len(held) != rows*seq {
			t.Fatalf("a row of %d holds %d", seq, len(held))
		}
	}
	if !slices.Equal(ids, []int64{7, 8, 9, 4, 1, 1}) {
		t.Errorf("the shorter text was padded with %v", ids)
	}
	if !slices.Equal(mask, []int64{1, 1, 1, 1, 0, 0}) {
		t.Errorf("the padding is marked: %v", mask)
	}
}

// A text the tokenizer gave nothing for is still a text, and what pools its row
// divides by something.
func TestAnEmptyTextIsMarkedAtOneToken(t *testing.T) {
	rows, seq, _, mask, _ := padBatch([][]int{{}}, 1)
	if rows != 1 || seq != 1 {
		t.Fatalf("one empty text was laid out %dx%d", rows, seq)
	}
	if !slices.Equal(mask, []int64{1}) {
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
