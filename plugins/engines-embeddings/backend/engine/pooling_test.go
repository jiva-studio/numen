package engine

import (
	"math"
	"slices"
	"testing"
)

// twoRows is the per-token output of two texts of three dimensions: the first holds three tokens and
// the second two, with a padding token after them.
func twoRows() (flat []float32, mask []int64) {
	flat = []float32{
		1, 0, 0, // row 0, token 0
		0, 3, 0, // row 0, token 1
		0, 0, 5, // row 0, token 2
		0, 2, 0, // row 1, token 0
		4, 0, 0, // row 1, token 1
		9, 9, 9, // row 1, padding
	}
	mask = []int64{1, 1, 1, 1, 1, 0}
	return flat, mask
}

func TestPoolKinds(t *testing.T) {
	flat, mask := twoRows()
	cases := []struct {
		kind string
		want [][]float32
	}{
		{PoolCLS, [][]float32{{1, 0, 0}, {0, 2, 0}}},
		{PoolLast, [][]float32{{0, 0, 5}, {4, 0, 0}}},
		{PoolMean, [][]float32{{1.0 / 3, 1, 5.0 / 3}, {2, 1, 0}}},
	}
	for _, tc := range cases {
		got, err := pool(tc.kind, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, false)
		if err != nil {
			t.Fatalf("%s: %v", tc.kind, err)
		}
		for row := range tc.want {
			if !isNear(got[row], tc.want[row]) {
				t.Errorf("%s row %d = %v, want %v", tc.kind, row, got[row], tc.want[row])
			}
		}
	}
}

func TestPoolNormalisesOnlyWhereTheProfileAsksForIt(t *testing.T) {
	flat, mask := twoRows()
	raw, err := pool(PoolLast, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, false)
	if err != nil {
		t.Fatal(err)
	}
	if raw[0][2] != 5 {
		t.Errorf("a vector that is not normalised is scaled: %v", raw[0])
	}
	unit, err := pool(PoolLast, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(unit[0], []float32{0, 0, 1}) {
		t.Errorf("a normalised vector is %v, want unit length", unit[0])
	}
}

func TestPoolOutputIsNoPerTokenPooling(t *testing.T) {
	flat, mask := twoRows()
	if _, err := pool(PoolOutput, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, true); err == nil {
		t.Error("output pooling of a per-token tensor gave no error")
	}
}

func TestSplitRowsKeepsTheVectorsOfAnOutputThatIsPooled(t *testing.T) {
	flat := []float32{3, 4, 0, 6}
	got, err := splitRows(flat, 2, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(got[0], []float32{0.6, 0.8}) || !isNear(got[1], []float32{0, 1}) {
		t.Errorf("normalised rows = %v", got)
	}
	raw, err := splitRows(flat, 2, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(raw[0], []float32{3, 4}) {
		t.Errorf("raw rows = %v", raw)
	}
	if _, err := splitRows(flat, 3, 2, true); err == nil {
		t.Error("a wrong number of values gave no error")
	}
}

func TestPadBatchSetsPositionsOfTheRealTokensAndZeroForPadding(t *testing.T) {
	paddings := newPaddingPool()
	rows, seq, in := padBatch(paddings, [][]int{{7, 8, 9}, {5}}, 0, 512)
	if rows != 2 || seq != 64 {
		t.Fatalf("rows, seq = %d, %d", rows, seq)
	}
	if got := in.positions[:4]; !slices.Equal(got, []int64{0, 1, 2, 0}) {
		t.Errorf("positions of row 0 = %v", got)
	}
	if got := in.positions[seq : seq+3]; !slices.Equal(got, []int64{0, 0, 0}) {
		t.Errorf("positions of row 1 = %v", got)
	}
	paddings.Put(in)
	_, _, again := padBatch(paddings, [][]int{{1}}, 0, 512)
	if again.positions[1] != 0 {
		t.Errorf("a reused padding kept old positions: %v", again.positions[:3])
	}
}

func TestChooseOutput(t *testing.T) {
	cases := []struct {
		name    string
		named   []string
		pooling string
		want    string
		isError bool
	}{
		{"per-token output of a mean model", []string{"last_hidden_state"}, PoolMean, "last_hidden_state", false},
		{"both, pooled by the model", []string{"last_hidden_state", "sentence_embedding"}, PoolOutput, "sentence_embedding", false},
		{"both, pooled here", []string{"sentence_embedding", "last_hidden_state"}, PoolLast, "last_hidden_state", false},
		{"only a sentence embedding", []string{"sentence_embedding"}, PoolMean, "sentence_embedding", false},
		{"output pooling without it", []string{"last_hidden_state", "other"}, PoolOutput, "", true},
		{"one unknown output", []string{"logits"}, PoolOutput, "logits", false},
		{"none we know", []string{"a", "b"}, PoolMean, "", true},
	}
	for _, tc := range cases {
		got, err := chooseOutput(tc.named, tc.pooling)
		if (err != nil) != tc.isError || got != tc.want {
			t.Errorf("%s: chooseOutput = %q, %v", tc.name, got, err)
		}
	}
}

func TestCheckInputs(t *testing.T) {
	base := []string{inputIDs, attentionMask}
	cases := []struct {
		name    string
		names   []string
		extra   []string
		want    modelInputs
		isError bool
	}{
		{"plain", base, nil, modelInputs{}, false},
		{"typed", append(slices.Clone(base), tokenTypeIDs), nil, modelInputs{isTyped: true}, false},
		{"positions asked and taken", append(slices.Clone(base), InputPositionIDs), []string{InputPositionIDs}, modelInputs{isPositioned: true}, false},
		{"positions taken but not asked", append(slices.Clone(base), InputPositionIDs), nil, modelInputs{}, true},
		{"positions asked but not taken", base, []string{InputPositionIDs}, modelInputs{}, true},
		{"cache inputs", append(slices.Clone(base), "past_key_values.0.key"), nil, modelInputs{}, true},
		{"no mask", []string{inputIDs}, nil, modelInputs{}, true},
	}
	for _, tc := range cases {
		got, err := checkInputs("model", tc.names, tc.extra)
		if (err != nil) != tc.isError || got != tc.want {
			t.Errorf("%s: checkInputs = %+v, %v", tc.name, got, err)
		}
	}
}

func TestGetPoolingMapsHeadToCLS(t *testing.T) {
	for in, want := range map[string]string{PoolHead: PoolCLS, PoolMean: PoolMean, PoolLast: PoolLast, PoolOutput: PoolOutput} {
		if got := getPooling(in); got != want {
			t.Errorf("getPooling(%q) = %q, want %q", in, got, want)
		}
	}
}

func isNear(got, want []float32) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > 1e-5 {
			return false
		}
	}
	return true
}
