package tokenizer_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/tokenizer"
)

// envTestModels names the folder that holds the files of the models: one folder per model, named by
// its repository id with the slash written as two dashes, holding tokenizer.json.
const envTestModels = "NUMEN_TEST_MODELS"

func getParityModels() []string {
	// The Unigram models of the profiles of engines-embeddings and engines-reranking, at the revisions
	// those profiles pin. testdata/parity/record.py records their reference ids.
	return []string{
		"Xenova/multilingual-e5-small",
		"Xenova/bge-m3",
		"onnx-community/gte-multilingual-reranker-base",
		"onnx-community/bge-reranker-v2-m3-ONNX",
	}
}

type parityFixture struct {
	Tokenizers string `json:"tokenizers"`
	Repo       string `json:"repo"`
	Revision   string `json:"revision"`
	SHA256     string `json:"sha256"`
	Singles    []struct {
		Text string `json:"text"`
		IDs  []int  `json:"ids"`
	} `json:"singles"`
	Pairs []struct {
		First     string `json:"first"`
		Second    string `json:"second"`
		MaxTokens int    `json:"maxTokens"`
		IDs       []int  `json:"ids"`
		TypeIDs   []int  `json:"typeIds"`
	} `json:"pairs"`
}

func readParityFixture(t *testing.T, folder string) parityFixture {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "parity", folder+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture parityFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Singles) == 0 || len(fixture.Pairs) == 0 {
		t.Fatalf("%s.json holds no case: a comparison against nothing", folder)
	}
	return fixture
}

// TestUnigramModelsMatchHuggingFaceTokenizers compares the ids of each model with the ids Python
// `tokenizers` gave for the same tokenizer.json: single texts without truncation, and pairs
// truncated longest_first. A model whose files are absent is skipped, and nothing is downloaded.
func TestUnigramModelsMatchHuggingFaceTokenizers(t *testing.T) {
	root := os.Getenv(envTestModels)
	if root == "" {
		t.Skipf("set %s to a folder of model files to run the parity test", envTestModels)
	}
	for _, repo := range getParityModels() {
		t.Run(repo, func(t *testing.T) {
			folder := strings.ReplaceAll(repo, "/", "--")
			path := filepath.Join(root, folder, "tokenizer.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("%s holds no tokenizer.json of %s", root, repo)
			}
			fixture := readParityFixture(t, folder)
			if sum := sha256.Sum256(raw); hex.EncodeToString(sum[:]) != fixture.SHA256 {
				t.Fatalf("%s is not the file the reference was recorded from (sha256 %s)", path, fixture.SHA256)
			}
			tok := loadTokenizer(t, path)
			checkParitySingles(t, tok, fixture)
			checkParityPairs(t, tok, fixture)
		})
	}
}

func checkParitySingles(t *testing.T, tok *tokenizer.Tokenizer, fixture parityFixture) {
	t.Helper()
	for _, one := range fixture.Singles {
		if got := tok.Encode(one.Text); !slices.Equal(got, one.IDs) {
			t.Errorf("Encode(%q)\n got %v\nwant %v", shorten(one.Text), got, one.IDs)
		}
	}
}

func checkParityPairs(t *testing.T, tok *tokenizer.Tokenizer, fixture parityFixture) {
	t.Helper()
	for _, one := range fixture.Pairs {
		got, err := tok.EncodePair(one.First, one.Second, one.MaxTokens)
		if err != nil {
			t.Errorf("EncodePair(%q, %q, %d): %v", shorten(one.First), shorten(one.Second), one.MaxTokens, err)
			continue
		}
		if !slices.Equal(got.IDs, one.IDs) || !slices.Equal(got.TypeIDs, one.TypeIDs) {
			t.Errorf("EncodePair(%q, %q, %d)\n got %v %v\nwant %v %v", shorten(one.First), shorten(one.Second), one.MaxTokens, got.IDs, got.TypeIDs, one.IDs, one.TypeIDs)
		}
	}
}

func shorten(text string) string {
	if runes := []rune(text); len(runes) > 60 {
		return string(runes[:60]) + "…"
	}
	return text
}
