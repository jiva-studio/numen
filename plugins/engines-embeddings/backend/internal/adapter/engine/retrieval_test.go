package engine

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine/onnx"
)

// envTestModels names the folder that holds the files of the curated models for the retrieval test: one
// folder per model, named by its id with the slash written as two dashes, laid out as the repository is
// (onnx/<file>, the data file beside it, and tokenizer.json). NUMEN_ONNX_RUNTIME_LIB names the library.
const envTestModels = "NUMEN_TEST_MODELS"

// minRankedFirstTwo is how many of the queries of the fixture must find their passage first or second.
const minRankedFirstTwo = 14

type retrievalFixture struct {
	Docs []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"docs"`
	Queries []struct {
		Query  string `json:"query"`
		Target string `json:"target"`
	} `json:"queries"`
}

// TestCuratedModelsFindThePassageOfAQuery loads each curated model that this machine holds the files of,
// embeds a fixture of invented passages and queries through the engine, and checks the width of the
// vectors and that most queries rank their passage first or second. A model whose files are absent is
// skipped, and nothing is downloaded.
func TestCuratedModelsFindThePassageOfAQuery(t *testing.T) {
	root := os.Getenv(envTestModels)
	if root == "" {
		t.Skipf("set %s to a folder of model files to run the retrieval test", envTestModels)
	}
	lib := onnx.GetLibrary()
	if lib == "" {
		t.Skipf("set %s to the ONNX Runtime library", onnx.EnvLibrary)
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "retrieval.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture retrievalFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, profile := range getShippedProfiles(t).GetCurated() {
		t.Run(profile.ID, func(t *testing.T) {
			dir := filepath.Join(root, strings.ReplaceAll(profile.ID, "/", "--"))
			if _, err := os.Stat(dir); err != nil {
				t.Skipf("%s holds no files of %s", root, profile.ID)
			}
			checkRetrieval(t, profile, dir, lib, fixture)
		})
	}
}

func checkRetrieval(t *testing.T, profile model.Profile, dir, lib string, fixture retrievalFixture) {
	t.Helper()
	info, local := model.NewConfig(profile)
	local.Dir, local.ShouldDownload, local.Runtime = flatten(t, dir), false, lib
	local.Threads, local.BatchTexts = 4, 8
	emb, err := Open(t.Context(), onnx.NewLibrary(), info, local, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if err := emb.Close(); err != nil {
			t.Error(err)
		}
	}()

	docs := make([]string, len(fixture.Docs))
	for i, d := range fixture.Docs {
		docs[i] = d.Text
	}
	queries := make([]string, len(fixture.Queries))
	for i, q := range fixture.Queries {
		queries[i] = q.Query
	}
	docVectors, err := emb.Embed(t.Context(), info.Handling.Apply(model.RoleDocument, docs))
	if err != nil {
		t.Fatalf("embed documents: %v", err)
	}
	queryVectors, err := emb.Embed(t.Context(), info.Handling.Apply(model.RoleQuery, queries))
	if err != nil {
		t.Fatalf("embed queries: %v", err)
	}
	for i, v := range append(slices.Clone(docVectors), queryVectors...) {
		if len(v) != profile.Dimensions {
			t.Fatalf("vector %d has %d dimensions, the profile says %d", i, len(v), profile.Dimensions)
		}
		if profile.IsNormalized() && math.Abs(norm(v)-1) > 1e-3 {
			t.Fatalf("vector %d has length %v, want a unit vector", i, norm(v))
		}
	}

	firstTwo := 0
	for qi, q := range fixture.Queries {
		rank := 1
		var targetScore float64
		for di, d := range fixture.Docs {
			if d.ID == q.Target {
				targetScore = dot(queryVectors[qi], docVectors[di])
			}
		}
		for di, d := range fixture.Docs {
			if d.ID != q.Target && dot(queryVectors[qi], docVectors[di]) > targetScore {
				rank++
			}
		}
		if rank <= 2 {
			firstTwo++
		}
		t.Logf("%-40q target %s rank %d", q.Query, q.Target, rank)
	}
	t.Logf("%s: %d of %d queries rank their passage first or second", profile.ID, firstTwo, len(fixture.Queries))
	if firstTwo < minRankedFirstTwo {
		t.Errorf("%s: %d of %d queries rank their passage first or second, want at least %d",
			profile.ID, firstTwo, len(fixture.Queries), minRankedFirstTwo)
	}
}

func flatten(t *testing.T, dir string) string {
	t.Helper()
	flat := t.TempDir()
	entries, err := os.ReadDir(filepath.Join(dir, model.ModelFolder))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{model.TokenizerFile}
	for _, e := range entries {
		names = append(names, filepath.Join(model.ModelFolder, e.Name()))
	}
	for _, name := range names {
		if err := os.Symlink(filepath.Join(dir, name), filepath.Join(flat, filepath.Base(name))); err != nil {
			t.Fatal(err)
		}
	}
	return flat
}

func dot(a, b []float32) float64 {
	var sum float64
	for i := range a {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}

func norm(v []float32) float64 { return math.Sqrt(dot(v, v)) }
