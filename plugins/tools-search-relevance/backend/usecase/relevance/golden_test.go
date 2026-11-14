package relevance_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-search-relevance/backend/usecase/relevance"
)

// goldenFixture is a sample vault and the calibration the calibration of the core measured on it
// before that code moved into this plugin. The samples are the titles with their vectors and the
// stored vectors of the passages. The expected values are the output of the core's
// usecase/floor Calibrate on those samples, so the plugin has to measure the same floor. The host
// double compares a title with a passage the way the core does, over the stored bytes.
type goldenFixture struct {
	Name   string `json:"name"`
	Titles []struct {
		Source int64     `json:"source"`
		Text   string    `json:"text"`
		Vector []float32 `json:"vector"`
	} `json:"titles"`
	Chunks []struct {
		Source int64  `json:"source"`
		Vector []byte `json:"vector"`
	} `json:"chunks"`
	Expected struct {
		Outcome    string  `json:"outcome"`
		Floor      float64 `json:"floor"`
		Median     float64 `json:"median"`
		Spread     float64 `json:"spread"`
		Histogram  []int   `json:"histogram"`
		QueryCount int     `json:"queryCount"`
		ChunkCount int     `json:"chunkCount"`
	} `json:"expected"`
}

func readFixtures(t *testing.T) []goldenFixture {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no golden fixture in testdata: %v", err)
	}
	fixtures := make([]goldenFixture, 0, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fixture goldenFixture
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures
}

func sourcePath(source int64) string { return fmt.Sprintf("note-%05d.md", source) }

func coreCosine(query, stored []float32) float64 {
	if len(query) != len(stored) || len(query) == 0 {
		return 0
	}
	var dot0, dot1, dot2, dot3 float32
	var left0, left1, left2, left3 float32
	var right0, right1, right2, right3 int32

	n := len(stored)
	i := 0
	for ; i+3 < n; i += 4 {
		q0, s0 := query[i], int32(stored[i])
		q1, s1 := query[i+1], int32(stored[i+1])
		q2, s2 := query[i+2], int32(stored[i+2])
		q3, s3 := query[i+3], int32(stored[i+3])

		dot0 += float32(q0 * float32(s0))
		left0 += float32(q0 * q0)
		right0 += s0 * s0

		dot1 += float32(q1 * float32(s1))
		left1 += float32(q1 * q1)
		right1 += s1 * s1

		dot2 += float32(q2 * float32(s2))
		left2 += float32(q2 * q2)
		right2 += s2 * s2

		dot3 += float32(q3 * float32(s3))
		left3 += float32(q3 * q3)
		right3 += s3 * s3
	}

	dot := float64((dot0 + dot1) + (dot2 + dot3))
	remaining := float64((left0 + left1) + (left2 + left3))
	right := float64((right0 + right1) + (right2 + right3))

	for ; i < n; i++ {
		q, s := query[i], int32(stored[i])
		dot += float64(q * float32(s))
		remaining += float64(q * q)
		right += float64(s * s)
	}

	if remaining == 0 || right == 0 {
		return 0
	}
	return dot / math.Sqrt(remaining*right)
}

func hostOf(fixture goldenFixture) (*hosttest.Search, *hosttest.Vault) {
	search := hosttest.NewSearch("physics")
	search.SetModel(hosttest.SearchModel{Name: modelName, Recipe: modelRecipe})
	vault := hosttest.New("physics")
	vectors := map[string][]float32{}
	for _, one := range fixture.Titles {
		vectors[one.Text] = one.Vector
		vault.File(sourcePath(one.Source), fmt.Sprintf("---\ntitle: %s\n---\n", one.Text))
	}
	search.SetEmbedder(func(text string) []float32 { return vectors[text] })
	chunks := make([]hosttest.SampledChunk, len(fixture.Chunks))
	for i, one := range fixture.Chunks {
		values := make([]float32, len(one.Vector))
		for j, b := range one.Vector {
			values[j] = float32(int8(b))
		}
		chunks[i] = hosttest.SampledChunk{Path: sourcePath(one.Source), Vector: values}
	}
	search.SetSampledChunks(modelRecipe, chunks)
	search.SetCosine(coreCosine)
	return search, vault
}

func TestTheSamplesOfTheCoreGiveTheSameFloorInThePlugin(t *testing.T) {
	for _, fixture := range readFixtures(t) {
		t.Run(fixture.Name, func(t *testing.T) {
			search, vault := hostOf(fixture)
			state, err := search.GetEmbeddingState(t.Context())
			if err != nil {
				t.Fatal(err)
			}

			outcome, got, err := relevance.NewCalibrator(newClock()).Calibrate(t.Context(), search, vault, state)
			if err != nil {
				t.Fatal(err)
			}
			want := fixture.Expected
			if want.Outcome == "too-few" {
				if outcome != relevance.TooFew {
					t.Fatalf("outcome %v, want TooFew", outcome)
				}
				return
			}
			if outcome != relevance.Calibrated {
				t.Fatalf("outcome %v, want Calibrated", outcome)
			}
			const tolerance = 1e-9
			if math.Abs(got.Floor-want.Floor) > tolerance || math.Abs(got.Median-want.Median) > tolerance ||
				math.Abs(got.Spread-want.Spread) > tolerance {
				t.Errorf("floor %v median %v spread %v, want %v, %v and %v",
					got.Floor, got.Median, got.Spread, want.Floor, want.Median, want.Spread)
			}
			if !slices.Equal(got.Histogram, want.Histogram) {
				t.Errorf("histogram %v, want %v", got.Histogram, want.Histogram)
			}
			if got.QueryCount != want.QueryCount || got.ChunkCount != want.ChunkCount {
				t.Errorf("measured %d queries against %d passages, want %d against %d",
					got.QueryCount, got.ChunkCount, want.QueryCount, want.ChunkCount)
			}
		})
	}
}

func TestTheFloorOfTheCoreIsStoredByTheFloorsOfThePlugin(t *testing.T) {
	for _, fixture := range readFixtures(t) {
		if fixture.Expected.Outcome == "too-few" {
			continue
		}
		t.Run(fixture.Name, func(t *testing.T) {
			search, vault := hostOf(fixture)
			search.SetProgress(10, 10)
			floors := relevance.New(relevance.Host{Search: searchOf(search), Vaults: func(id domain.VaultID) host.NoteLookup { return vault.With(id) }, Store: foldersOf(vault)}, newClock())

			outcome, state, err := floors.Calibrate(t.Context(), "")
			if err != nil || outcome != relevance.Calibrated {
				t.Fatalf("got %v, %v", outcome, err)
			}
			stored, found := search.StoredFloor(modelRecipe)
			if !found || math.Abs(stored-fixture.Expected.Floor) > 1e-9 {
				t.Errorf("the host stores %v (%v), want %v", stored, found, fixture.Expected.Floor)
			}
			if state.Source != relevance.SourceCalibrated || math.Abs(state.Floor-fixture.Expected.Floor) > 1e-9 {
				t.Errorf("state %+v, want the calibrated floor %v", state, fixture.Expected.Floor)
			}
		})
	}
}
