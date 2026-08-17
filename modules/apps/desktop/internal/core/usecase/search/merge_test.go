package search

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// ranking is chunk numbers in the order a half returned them.
func ranking(chunks ...int64) []domain.Passage {
	out := make([]domain.Passage, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, domain.Passage{Chunk: c, Source: "s"})
	}
	return out
}

func numbers(passages []domain.Passage) []int64 {
	out := make([]int64, 0, len(passages))
	for _, p := range passages {
		out = append(out, p.Chunk)
	}
	return out
}

func TestTheFusedOrderIsTheSumOfReciprocalRanks(t *testing.T) {
	// The formula is pinned here, because everything above it is unchanged by a
	// merge that ranks differently and nothing else notices.
	for _, c := range []struct {
		name  string
		words []int64
		dense []int64
		want  []int64
	}{
		{
			name:  "one ranking is passed through",
			words: []int64{1, 2, 3},
			want:  []int64{1, 2, 3},
		},
		{
			// 2 scores 1/62 + 1/61, above 1's single 1/61.
			name:  "agreed second beats a lone first",
			words: []int64{1, 2},
			dense: []int64{2, 3},
			want:  []int64{2, 1, 3},
		},
		{
			// Both halves put it first: 2/61 against 1/61 for each of the rest.
			name:  "agreed first stays first",
			words: []int64{1, 2, 3},
			dense: []int64{1, 3, 2},
			want:  []int64{1, 2, 3},
		},
		{
			// 1 is 1/61 + 1/64 = 0.03202, above 5 at 1/65 + 1/61 = 0.03178: the
			// half that placed each of them first is outweighed by where the
			// other half put it.
			name:  "both halves count, and where the second one placed it decides",
			words: []int64{1, 2, 3, 4, 5},
			dense: []int64{5, 6, 7, 1},
			want:  []int64{1, 5, 2, 6, 3, 7, 4},
		},
		{
			// Nothing agrees, so both halves' first places tie and the chunk
			// number decides.
			name:  "a tie is broken by the row, so one index gives one answer",
			words: []int64{9},
			dense: []int64{4},
			want:  []int64{4, 9},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var rankings [][]domain.Passage
			rankings = append(rankings, ranking(c.words...))
			if c.dense != nil {
				rankings = append(rankings, ranking(c.dense...))
			}
			got := numbers(merge(rankings...))
			if !slices.Equal(got, c.want) {
				t.Errorf("fused order %v, want %v", got, c.want)
			}
		})
	}
}

func TestOneDocumentTakesOneResult(t *testing.T) {
	// One passage per source, so an answer names each document once.
	fused := []domain.Passage{
		{Chunk: 1, Source: "book.epub"},
		{Chunk: 2, Source: "book.epub"},
		{Chunk: 3, Source: "note.md"},
		{Chunk: 4, Source: "book.epub"},
	}
	got := collapse(fused, 10)
	if len(got) != 2 {
		t.Fatalf("%d results, want one per document: %+v", len(got), got)
	}
	if got[0].Chunk != 1 || got[1].Chunk != 3 {
		t.Errorf("the best-ranked chunk of each document is not what survived: %+v", got)
	}
}

func TestTheLimitIsWhatComesBack(t *testing.T) {
	fused := []domain.Passage{
		{Chunk: 1, Source: "a"},
		{Chunk: 2, Source: "b"},
		{Chunk: 3, Source: "c"},
	}
	if got := collapse(fused, 2); len(got) != 2 {
		t.Errorf("%d results for a limit of two: %+v", len(got), got)
	}
}
