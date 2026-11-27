package engine

import (
	"encoding/json"
	"fmt"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"context"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-reranking/backend/domain/profile"
)

// envTestModels names the folder that holds the files of the shipped models: one folder per model,
// named by its id with the slash written as two dashes, laid out as the repository is (onnx/<file> and
// tokenizer.json). NUMEN_ONNX_RUNTIME_LIB names the library. Without it the tests skip, so CI skips them.
const envTestModels = "NUMEN_TEST_MODELS"

func getTestModels(t *testing.T) string {
	t.Helper()
	root := os.Getenv(envTestModels)
	if root == "" {
		t.Skipf("set %s to a folder of model files to run the tests of the shipped models", envTestModels)
	}
	return root
}

func getShippedProfiles(t *testing.T) []profile.Profile {
	t.Helper()
	table, err := profile.GetShippedProfiles()
	if err != nil {
		t.Fatal(err)
	}
	curated := table.GetCurated()
	if len(curated) != 2 {
		t.Fatalf("%d shipped profiles, want bge and gte", len(curated))
	}
	return curated
}

func getModelFolder(id string) string {
	return strings.ReplaceAll(id, "/", "--")
}

type referencePair struct {
	Query    string `json:"query"`
	Document string `json:"document"`
	IDs      []int  `json:"ids"`
	TypeIDs  []int  `json:"typeIds"`
}

// TestTokenizerMatchesReferenceIDs encodes each pair of testdata/tokens/<model folder>.json as the
// scorer does and compares the ids with the ones Python `tokenizers` gave (testdata/tokens/record.py).
func TestTokenizerMatchesReferenceIDs(t *testing.T) {
	root := getTestModels(t)
	for _, p := range getShippedProfiles(t) {
		t.Run(p.ID, func(t *testing.T) {
			path := filepath.Join(root, getModelFolder(p.ID), "tokenizer.json")
			if _, err := os.Stat(path); err != nil {
				t.Skipf("%s holds no tokenizer.json of %s", root, p.ID)
			}
			raw, err := os.ReadFile(filepath.Join("testdata", "tokens", getModelFolder(p.ID)+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var reference []referencePair
			if err := json.Unmarshal(raw, &reference); err != nil {
				t.Fatal(err)
			}
			if len(reference) == 0 {
				t.Fatal("the reference holds no pair")
			}
			pairs, err := loadPairTokenizer(path, p.MaxTokens)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range reference {
				got, err := pairs.encode(want.Query, want.Document)
				if err != nil {
					t.Errorf("encode(%q, %q): %v", shorten(want.Query), shorten(want.Document), err)
					continue
				}
				if !slices.Equal(got.IDs, want.IDs) || !slices.Equal(got.TypeIDs, want.TypeIDs) {
					t.Errorf("encode(%q, %q)\n got %v %v\nwant %v %v", shorten(want.Query), shorten(want.Document), got.IDs, got.TypeIDs, want.IDs, want.TypeIDs)
				}
			}
		})
	}
}

func shorten(text string) string {
	if runes := []rune(text); len(runes) > 40 {
		return string(runes[:40]) + "…"
	}
	return text
}

type relevanceFixture []struct {
	Query     string   `json:"query"`
	Relevant  string   `json:"relevant"`
	Unrelated []string `json:"unrelated"`
}

// TestShippedProfilesScoreARelevantDocumentAboveAnUnrelatedOne opens each shipped model from the files
// of NUMEN_TEST_MODELS, laid out as the download cache keeps them at the pinned revision, with fetching
// forbidden, so nothing is downloaded.
func TestShippedProfilesScoreARelevantDocumentAboveAnUnrelatedOne(t *testing.T) {
	root := getTestModels(t)
	if onnx.GetLibrary() == "" {
		t.Skipf("set %s to the ONNX Runtime library", onnx.EnvLibrary)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "rerank.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture relevanceFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, p := range getShippedProfiles(t) {
		t.Run(p.ID, func(t *testing.T) {
			dir := filepath.Join(root, getModelFolder(p.ID))
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p.OnnxFile))); err != nil {
				t.Skipf("%s holds no %s of %s", root, p.OnnxFile, p.ID)
			}
			fetch := Fetch{Dependency: func(_ context.Context, id string) (string, error) {
				for name, declared := range p.Dependencies {
					if declared == id {
						return filepath.Join(dir, filepath.FromSlash(name)), nil
					}
				}
				return "", fmt.Errorf("%s declares no %s", p.ID, id)
			}}
			scorer, err := Open(t.Context(), onnx.NewLibrary(), p, fetch, hosttest.NewQueue().EnqueueWork)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer func() {
				if err := scorer.Close(); err != nil {
					t.Error(err)
				}
			}()
			for _, one := range fixture {
				checkRelevance(t, scorer, one.Query, append([]string{one.Relevant}, one.Unrelated...))
			}
		})
	}
}

func checkRelevance(t *testing.T, scorer *Scorer, query string, documents []string) {
	t.Helper()
	scores, err := scorer.ScorePairs(t.Context(), query, documents)
	if err != nil {
		t.Fatalf("ScorePairs(%q): %v", query, err)
	}
	if len(scores) != len(documents) {
		t.Fatalf("%q: %d scores for %d documents", query, len(scores), len(documents))
	}
	t.Logf("%-50q %v", query, scores)
	for i, score := range scores {
		if score <= 0 || score >= 1 {
			t.Errorf("%q: score %d = %v, want one in (0, 1)", query, i, score)
		}
	}
	for i := 1; i < len(scores); i++ {
		if scores[0] <= scores[i] {
			t.Errorf("%q: the relevant document scores %v, the unrelated %q scores %v", query, scores[0], shorten(documents[i]), scores[i])
		}
	}
	reversed := slices.Clone(documents)
	slices.Reverse(reversed)
	again, err := scorer.ScorePairs(t.Context(), query, reversed)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(again)
	for i := range scores {
		if diff := scores[i] - again[i]; diff > 1e-3 || diff < -1e-3 {
			t.Errorf("%q: document %d scores %v in one order and %v in the other", query, i, scores[i], again[i])
		}
	}
}
