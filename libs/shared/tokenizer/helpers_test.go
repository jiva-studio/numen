package tokenizer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// fixturePath is the hand-made Unigram tokenizer.json the unit tests read. Its template gives <s>
// the id 5 and </s> the id 7, which are not the vocabulary positions of the two pieces.
const fixturePath = "testdata/unigram.json"

const (
	bosID = 5
	eosID = 7
	unkID = 0
	padID = 3
)

func readFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	return file
}

func writeVariant(t *testing.T, change func(file map[string]any)) string {
	t.Helper()
	file := readFixture(t)
	change(file)
	raw, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "tokenizer.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func getObject(t *testing.T, value any) map[string]any {
	t.Helper()
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%v is not a JSON object", value)
	}
	return object
}

func getArray(t *testing.T, value any) []any {
	t.Helper()
	array, ok := value.([]any)
	if !ok {
		t.Fatalf("%v is not a JSON array", value)
	}
	return array
}

func getModel(t *testing.T, file map[string]any) map[string]any {
	t.Helper()
	return getObject(t, file["model"])
}

func loadTokenizer(t *testing.T, path string) *tokenizer.Tokenizer {
	t.Helper()
	tok, err := tokenizer.Load(path)
	if err != nil {
		t.Fatalf("Load(%s): %v", path, err)
	}
	return tok
}

func loadFixture(t *testing.T) *tokenizer.Tokenizer {
	t.Helper()
	return loadTokenizer(t, fixturePath)
}

func getPieceIDs(t *testing.T, pieces ...string) []int {
	t.Helper()
	vocab := getArray(t, getModel(t, readFixture(t))["vocab"])
	ids := make([]int, 0, len(pieces))
	for _, piece := range pieces {
		id := slices.IndexFunc(vocab, func(entry any) bool { return getArray(t, entry)[0] == piece })
		if id < 0 {
			t.Fatalf("the fixture has no piece %q", piece)
		}
		ids = append(ids, id)
	}
	return ids
}

func getSingleIDs(t *testing.T, pieces ...string) []int {
	t.Helper()
	return slices.Concat([]int{bosID}, getPieceIDs(t, pieces...), []int{eosID})
}

func checkEncode(t *testing.T, tok *tokenizer.Tokenizer, text string, want []int) {
	t.Helper()
	if got := tok.Encode(text); !slices.Equal(got, want) {
		t.Errorf("Encode(%q) = %v, want %v", text, got, want)
	}
}
