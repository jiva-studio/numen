package relevance_test

import (
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest/searchtest"
)

const (
	modelName   = "test-model"
	modelRecipe = "test|test-model|4|0|mean|int8@0.4"
)

func getMeasuredAt() time.Time {
	return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
}

func newClock() *clocktest.Fake { return newFakeClock(getMeasuredAt()) }

func newHost(hint float64) (*searchtest.Search, *hosttest.Vault) {
	search := searchtest.NewSearch("physics", "chemistry")
	search.SetModel(searchtest.SearchModel{Name: modelName, Recipe: modelRecipe, FloorHint: hint})
	search.SetEmbedder(func(string) []float32 { return []float32{1, 0, 0, 0} })
	vault := hosttest.New("physics")
	vault.LinkBeside(hosttest.New("chemistry"))
	return search, vault
}

// maxNotes is the number of notes the helpers write at most.
const maxNotes = 100

func notePath(n int) string { return fmt.Sprintf("note-%d.md", n) }

func setNotes(vault *hosttest.Vault, n int) {
	for i := 1; i <= maxNotes; i++ {
		if i > n {
			vault.Remove(notePath(i))
			continue
		}
		vault.WriteFile(notePath(i), fmt.Sprintf("---\ntitle: title %d\n---\n", i-1))
	}
}

func sampledChunksOf(n, sources int) []searchtest.SampledChunk {
	out := make([]searchtest.SampledChunk, n)
	for i := range out {
		out[i] = searchtest.SampledChunk{Path: fmt.Sprintf("source-%d.md", 100+i%sources), Vector: []float32{64, 64, 64, 64}}
	}
	return out
}

func fill(search *searchtest.Search, vault *hosttest.Vault, notes int, chunks []searchtest.SampledChunk) {
	setNotes(vault, notes)
	search.SetSampledChunks(modelRecipe, chunks)
	search.SetProgress(int64(len(chunks)), int64(len(chunks)))
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

func newFakeClock(start time.Time) *clocktest.Fake {
	source := new(clocktest.Fake)
	source.Set(start)
	return source
}
