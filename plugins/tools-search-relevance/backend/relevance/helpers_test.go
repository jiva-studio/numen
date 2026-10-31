package relevance_test

import (
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

const (
	modelName   = "test-model"
	modelRecipe = "test|test-model|4|0|mean|int8@0.4"
)

var measuredAt = time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)

// newClock is a clock that reports measuredAt.
func newClock() *clock.Fake { return clock.NewFake(measuredAt) }

// newHost is a host that shows the vault "physics", knows "chemistry" and runs the test model, with
// every title embedded as the unit vector along the first dimension.
func newHost(hint float64) (*hosttest.Search, *hosttest.Vault) {
	search := hosttest.NewSearch("physics", "chemistry")
	search.SetModel(hosttest.SearchModel{Name: modelName, Recipe: modelRecipe, FloorHint: hint})
	search.SetEmbedder(func(string) []float32 { return []float32{1, 0, 0, 0} })
	vault := hosttest.New("physics")
	vault.Beside(hosttest.New("chemistry"))
	return search, vault
}

// stored is the stored form of a vector: a signed byte per dimension.
func stored(values ...int8) []byte {
	out := make([]byte, len(values))
	for i, value := range values {
		out[i] = byte(value)
	}
	return out
}

// titlesOf is n titles, each on a source of its own numbered from 1.
func titlesOf(n int) []host.NoteTitle {
	out := make([]host.NoteTitle, n)
	for i := range out {
		out[i] = host.NoteTitle{Source: int64(i + 1), Title: fmt.Sprintf("title %d", i)}
	}
	return out
}

// passagesOf is n passages spread over `sources` sources numbered from 100, each at 0.5 cosine
// similarity to the unit vector along the first dimension.
func passagesOf(n, sources int) []host.ChunkVector {
	out := make([]host.ChunkVector, n)
	for i := range out {
		out[i] = host.ChunkVector{Source: int64(100 + i%sources), Vector: stored(64, 64, 64, 64)}
	}
	return out
}

// fill makes the vault of the search hold enough embedded notes to measure.
func fill(search *hosttest.Search, titles int, passages []host.ChunkVector) {
	search.SetNoteTitles(titlesOf(titles))
	search.SetChunkVectors(modelRecipe, passages)
	search.SetProgress(int64(len(passages)), int64(len(passages)))
}

func sum(counts []int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
