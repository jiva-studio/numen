package search

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// ranking is chunks in the order a half returned them.
func ranking(chunks ...domain.ChunkID) []domain.Passage {
	out := make([]domain.Passage, 0, len(chunks))
	for _, c := range chunks {
		out = append(out, domain.Passage{ChunkID: c, Source: "s"})
	}
	return out
}

func chunksIn(passages []domain.Passage) []domain.ChunkID {
	out := make([]domain.ChunkID, 0, len(passages))
	for _, p := range passages {
		out = append(out, p.ChunkID)
	}
	return out
}

func TestTheFusedOrderIsTheSumOfReciprocalRanks(t *testing.T) {
	// The formula is pinned here, because everything above it is unchanged by a
	// merge that ranks differently and nothing else notices.
	for _, c := range []struct {
		name  string
		words []domain.ChunkID
		dense []domain.ChunkID
		want  []domain.ChunkID
	}{
		{
			name:  "one ranking is passed through",
			words: []domain.ChunkID{"1", "2", "3"},
			want:  []domain.ChunkID{"1", "2", "3"},
		},
		{
			// 2 scores 1/62 + 1/61, above 1's single 1/61.
			name:  "agreed second beats a lone first",
			words: []domain.ChunkID{"1", "2"},
			dense: []domain.ChunkID{"2", "3"},
			want:  []domain.ChunkID{"2", "1", "3"},
		},
		{
			// Both halves put it first: 2/61 against 1/61 for each of the rest.
			name:  "agreed first stays first",
			words: []domain.ChunkID{"1", "2", "3"},
			dense: []domain.ChunkID{"1", "3", "2"},
			want:  []domain.ChunkID{"1", "2", "3"},
		},
		{
			// 1 is 1/61 + 1/64 = 0.03202, above 5 at 1/65 + 1/61 = 0.03178: the
			// half that placed each of them first is outweighed by where the
			// other half put it.
			name:  "both halves count, and where the second one placed it decides",
			words: []domain.ChunkID{"1", "2", "3", "4", "5"},
			dense: []domain.ChunkID{"5", "6", "7", "1"},
			want:  []domain.ChunkID{"1", "5", "2", "6", "3", "7", "4"},
		},
		{
			// Nothing agrees, so both halves' first places tie and the chunk
			// decides.
			name:  "a tie is broken by the chunk, so one index gives one answer",
			words: []domain.ChunkID{"9"},
			dense: []domain.ChunkID{"4"},
			want:  []domain.ChunkID{"4", "9"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var rankings [][]domain.Passage
			rankings = append(rankings, ranking(c.words...))
			if c.dense != nil {
				rankings = append(rankings, ranking(c.dense...))
			}
			got := chunksIn(merge(rankings...))
			if !slices.Equal(got, c.want) {
				t.Errorf("fused order %v, want %v", got, c.want)
			}
		})
	}
}

func TestOneDocumentTakesOneResult(t *testing.T) {
	// One passage per source, so an answer names each document once.
	fused := []domain.Passage{
		{ChunkID: "1", Source: "book.epub"},
		{ChunkID: "2", Source: "book.epub"},
		{ChunkID: "3", Source: "note.md"},
		{ChunkID: "4", Source: "book.epub"},
	}
	got := collapse(fused, nil, 1, 10)
	if len(got) != 2 {
		t.Fatalf("%d results, want one per document: %+v", len(got), got)
	}
	if got[0].ChunkID != "1" || got[1].ChunkID != "3" {
		t.Errorf("the best-ranked chunk of each document is not what survived: %+v", got)
	}
}

func TestADocumentAnswersWithTheSectionNamedWhatWasAsked(t *testing.T) {
	// The paragraph that says the words most often stands first in the fused
	// order, and the section carrying the name is further down.
	fused := []domain.Passage{
		{ChunkID: "2", Source: "book.epub"},
		{ChunkID: "3", Source: "note.md"},
		{ChunkID: "1", Source: "book.epub"},
	}
	named := []domain.Passage{{ChunkID: "1", Source: "book.epub"}}

	got := collapse(fused, named, 1, 10)
	if len(got) != 2 {
		t.Fatalf("%d results, want one per document: %+v", len(got), got)
	}
	if got[0].ChunkID != "1" {
		t.Errorf("the book answered with chunk %s, want the section it names", got[0].ChunkID)
	}
	if got[0].Source != "book.epub" || got[1].Source != "note.md" {
		t.Errorf("the documents came back in another order: %+v", got)
	}
}

func TestADocumentWithNoSectionOfThatNameIsUnchanged(t *testing.T) {
	fused := []domain.Passage{
		{ChunkID: "2", Source: "book.epub"},
		{ChunkID: "1", Source: "book.epub"},
	}
	named := []domain.Passage{{ChunkID: "9", Source: "other.epub"}}

	got := collapse(fused, named, 1, 10)
	if len(got) != 1 || got[0].ChunkID != "2" {
		t.Errorf("got %+v, want the best-ranked chunk of the book", got)
	}
}

func TestADocumentAnswersWithAsManyPassagesAsWereAskedFor(t *testing.T) {
	fused := []domain.Passage{
		{ChunkID: "1", Source: "book.epub", Start: 0, Length: 10},
		{ChunkID: "2", Source: "book.epub", Start: 90, Length: 10},
		{ChunkID: "3", Source: "note.md", Start: 0, Length: 10},
		{ChunkID: "4", Source: "book.epub", Start: 500, Length: 10},
	}

	got := collapse(fused, nil, 2, 10)
	if len(got) != 3 {
		t.Fatalf("%d results, want two of the book and the note: %+v", len(got), got)
	}
	if got[0].ChunkID != "1" || got[1].ChunkID != "2" || got[2].ChunkID != "3" {
		t.Errorf("got %+v, want the book's two best and the note", got)
	}
}

func TestASectionIsTheFirstOfWhatADocumentAnswersWith(t *testing.T) {
	// The section stands lower in the fused order than the passage that says
	// the words most often, and both come back.
	fused := []domain.Passage{
		{ChunkID: "2", Source: "book.epub", Start: 90, Length: 10},
		{ChunkID: "1", Source: "book.epub", Start: 0, Length: 10},
	}
	named := []domain.Passage{{ChunkID: "1", Source: "book.epub", Start: 0, Length: 10}}

	got := collapse(fused, named, 2, 10)
	if len(got) != 2 {
		t.Fatalf("%d results, want the section and the passage: %+v", len(got), got)
	}
	if got[0].ChunkID != "1" || got[1].ChunkID != "2" {
		t.Errorf("got %+v, want the section first and the passage after it", got)
	}
}

func TestOnePlaceIsOnePassageHoweverManyRowsStandThere(t *testing.T) {
	// A hit and the chunk enclosing it are two rows saying the same place.
	fused := []domain.Passage{
		{ChunkID: "7", Source: "book.epub", Start: 100, Length: 20},
		{ChunkID: "8", Source: "book.epub", Start: 100, Length: 20},
		{ChunkID: "9", Source: "book.epub", Start: 400, Length: 20},
	}

	got := collapse(fused, nil, 3, 10)
	if len(got) != 2 {
		t.Fatalf("%d passages, want the two places: %+v", len(got), got)
	}
	if got[0].Start != 100 || got[1].Start != 400 {
		t.Errorf("got %+v", got)
	}
}

func TestTheLimitIsWhatComesBack(t *testing.T) {
	fused := []domain.Passage{
		{ChunkID: "1", Source: "a"},
		{ChunkID: "2", Source: "b"},
		{ChunkID: "3", Source: "c"},
	}
	if got := collapse(fused, nil, 1, 2); len(got) != 2 {
		t.Errorf("%d results for a limit of two: %+v", len(got), got)
	}
}
