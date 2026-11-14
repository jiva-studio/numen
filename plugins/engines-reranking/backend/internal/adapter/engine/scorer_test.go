package engine

import (
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func sigmoid(logit float64) float64 { return 1 / (1 + math.Exp(-logit)) }

func TestReadScoresAppliesSigmoid(t *testing.T) {
	scores, err := readScores([]float32{0, 2.5, -1.25}, []int64{3, 1})

	if err != nil {
		t.Fatal(err)
	}
	want := []float64{0.5, sigmoid(2.5), sigmoid(-1.25)}
	if len(scores) != len(want) {
		t.Fatalf("scores = %v, want %v", scores, want)
	}
	for i := range want {
		if math.Abs(float64(scores[i])-want[i]) > 1e-6 {
			t.Errorf("score %d = %v, want %v", i, scores[i], want[i])
		}
	}
}

func TestReadScoresRejectsMoreThanOneLogit(t *testing.T) {
	for name, shape := range map[string][]int64{
		"two labels":    {2, 2},
		"three labels":  {1, 3},
		"no label axis": {4},
	} {
		if _, err := readScores([]float32{0.1, 0.2, 0.3, 0.4}[:shapeSize(shape)], shape); err == nil {
			t.Errorf("%s: readScores accepted the shape %v", name, shape)
		}
	}
}

func TestReadScoresRejectsLogitsThatDoNotFitTheShape(t *testing.T) {
	if _, err := readScores([]float32{0.1, 0.2}, []int64{3, 1}); err == nil {
		t.Error("readScores accepted 2 logits for 3 rows")
	}
}

func shapeSize(shape []int64) int {
	size := 1
	for _, n := range shape {
		size *= int(n)
	}
	return size
}

func TestOpenRejectsAGraphWithAnUnknownInput(t *testing.T) {
	_, err := checkInputs("onnx-community/lantern-reranker", []string{inputIDs, attentionMask, "past_key_values.0.key"})

	if err == nil || !strings.Contains(err.Error(), "past_key_values.0.key") {
		t.Errorf("checkInputs error = %v, want one that names the input", err)
	}
}

func TestCheckInputsAcceptsTheTokensTheMaskAndTheTokenTypes(t *testing.T) {
	plain, err := checkInputs("owner/lantern-reranker", []string{inputIDs, attentionMask})
	if err != nil || plain.isTyped {
		t.Errorf("plain inputs = %+v, %v", plain, err)
	}

	typed, err := checkInputs("owner/lantern-reranker", []string{inputIDs, attentionMask, tokenTypeIDs})
	if err != nil || !typed.isTyped {
		t.Errorf("typed inputs = %+v, %v", typed, err)
	}

	for _, names := range [][]string{{inputIDs}, {attentionMask}} {
		if _, err := checkInputs("owner/lantern-reranker", names); err == nil {
			t.Errorf("checkInputs accepted %v", names)
		}
	}
}

func TestLoadPairTokenizerRejectsAFileWithoutAPairTemplate(t *testing.T) {
	_, err := loadPairTokenizer(filepath.Join("testdata", "no-pair-template.json"), 512)

	if err == nil || !strings.Contains(err.Error(), "tokenizer.json has no pair template") {
		t.Errorf("loadPairTokenizer error = %v, want tokenizer.json has no pair template", err)
	}
}

func TestEncodeCutsAPairToTheTokensOfTheProfile(t *testing.T) {
	pairs, err := loadPairTokenizer(filepath.Join("testdata", "pair-template.json"), 9)
	if err != nil {
		t.Fatal(err)
	}

	long := strings.Repeat("hello world abc ", 20)
	enc, err := pairs.encode(long, long)

	if err != nil {
		t.Fatal(err)
	}
	if len(enc.IDs) != 9 || len(enc.TypeIDs) != 9 {
		t.Errorf("encoded %d ids and %d type ids, want 9 of each", len(enc.IDs), len(enc.TypeIDs))
	}
	if enc.IDs[0] != 5 || enc.IDs[len(enc.IDs)-1] != 7 || slices.Index(enc.TypeIDs, 1) < 0 {
		t.Errorf("encoding %v %v is not <s> A </s></s> B </s> of the pair template", enc.IDs, enc.TypeIDs)
	}
}
