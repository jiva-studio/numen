package note_test

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// A note is cut at the sizes given, and the chunk that carries a vector stays
// under the limit. A chunk over the model's input limit is embedded truncated,
// and then it indexes text it does not hold.
func TestASmallChunkIsCutUnderTheLimitGiven(t *testing.T) {
	body := strings.TrimSpace(strings.Repeat("chunks carry vectors ", 200))
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/cut.md", Size: int64(len(body)), ModTime: time.Unix(0, 1)},
		Title:       "Cut",
		Body:        body,
	}

	for _, limit := range []int{64, 512} {
		cut := note.CutNotes{Sizes: chunking.Sizes{Limit: limit}}.Execute([]domain.Note{n})
		chunks := cut[0].Chunks
		if len(chunks) != 1 {
			t.Fatalf("a note is one large chunk, and it was cut into %d", len(chunks))
		}
		if chunks[0].Length != len(body) {
			t.Errorf("the large chunk holds %d bytes of a note of %d", chunks[0].Length, len(body))
		}
		if len(chunks[0].Small) == 0 {
			t.Fatalf("nothing inside the note carries a vector at a limit of %d", limit)
		}
		for _, small := range chunks[0].Small {
			if held := utf8.RuneCountInString(small.Text); held > limit {
				t.Errorf("a chunk holds %d characters, told to hold %d", held, limit)
			}
		}
	}
}

// A deck keeps its sections and its cards without the mark they end in, and
// has no chunks; a stencil has neither.
func TestADeckIsKeptByItsHeadingsAndNotCut(t *testing.T) {
	deck := domain.Note{
		Type: domain.TypeDeck, Title: "Deck", Body: "# Part\n## Q?? \n",
		Headings: []domain.Heading{{Level: 1, Text: "Part"}, {Level: 2, Text: "Q"}, {Level: 3, Text: "Front"}},
	}
	stencil := domain.Note{Type: domain.TypeStencil, Title: "S", Headings: []domain.Heading{{Level: 1, Text: "Face"}}}

	cut := note.CutNotes{}.Execute([]domain.Note{deck, stencil})
	if len(cut[0].Chunks) != 0 || len(cut[1].Chunks) != 0 {
		t.Errorf("a deck or a stencil was cut: %+v", cut)
	}
	if len(cut[0].Headings) != 2 {
		t.Errorf("a deck keeps its section and its card, got %+v", cut[0].Headings)
	}
	if len(cut[1].Headings) != 0 {
		t.Errorf("a stencil keeps no heading, got %+v", cut[1].Headings)
	}
}
