package relevance_test

import (
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

const (
	modelName   = "test-model"
	modelRecipe = "test|test-model|4|0|mean|int8@0.4"
)

// getMeasuredAt returns the instant every measurement of these tests is taken at.
func getMeasuredAt() time.Time {
	return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
}

// newClock is a clock that reports getMeasuredAt().
func newClock() *clock.Fake { return clock.NewFake(getMeasuredAt()) }

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

// maxNotes is the number of notes the helpers write at most.
const maxNotes = 100

// notePath is the path of the nth note, numbered from 1.
func notePath(n int) string { return fmt.Sprintf("note-%d.md", n) }

// setNotes makes the vault hold n notes titled "title 0" to "title n-1", and no other.
func setNotes(vault *hosttest.Vault, n int) {
	for i := 1; i <= maxNotes; i++ {
		if i > n {
			vault.Remove(notePath(i))
			continue
		}
		vault.File(notePath(i), fmt.Sprintf("---\ntitle: title %d\n---\n", i-1))
	}
}

// passagesOf is n passages spread over `sources` notes named "source-100.md" onwards, each at 0.5
// cosine similarity to the unit vector along the first dimension.
func passagesOf(n, sources int) []hosttest.SearchPassage {
	out := make([]hosttest.SearchPassage, n)
	for i := range out {
		out[i] = hosttest.SearchPassage{Path: fmt.Sprintf("source-%d.md", 100+i%sources), Vector: []float32{64, 64, 64, 64}}
	}
	return out
}

// fill makes the vault hold the notes and the search the passages, enough to measure.
func fill(search *hosttest.Search, vault *hosttest.Vault, notes int, passages []hosttest.SearchPassage) {
	setNotes(vault, notes)
	search.SetPassages(modelRecipe, passages)
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
