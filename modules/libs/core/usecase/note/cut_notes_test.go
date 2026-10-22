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
func TestASmallChunkIsSplitBelowTheLimitGiven(t *testing.T) {
	body := strings.TrimSpace(strings.Repeat("chunks carry vectors ", 200))
	n := domain.Note{
		Fingerprint: domain.Fingerprint{Path: "notes/cut.md", Size: int64(len(body)), ModTime: time.Unix(0, 1)},
		Title:       "Cut",
		Body:        body,
	}

	for _, limit := range []int{64, 512} {
		cutResult := note.CutNotes{Sizes: chunking.Sizes{Limit: limit}}.Execute([]domain.Note{n})
		chunks := cutResult[0].Chunks
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
			if target := utf8.RuneCountInString(small.Text); target > limit {
				t.Errorf("a chunk holds %d characters, instructed to hold %d", target, limit)
			}
		}
	}
}

// A type a plugin declares is kept as its profile says: here only the headings of two levels, the
// second without the mark it ends in, and no chunks; a type with no headings keeps none.
func TestANoteOfADeclaredTypeIsCachedAsItsProfileReturns(t *testing.T) {
	profiles := map[domain.NoteType]domain.NoteProfile{
		"sheet": {HasChunks: true, HeadingLevels: []int{1, 2}, MarkedLevels: []int{2}},
		"bare":  {HeadingLevels: []int{}},
	}
	cuts := note.CutNotes{Profiles: func(t domain.NoteType) (domain.NoteProfile, bool) {
		profile, declared := profiles[t]
		return profile, declared
	}}
	sheet := domain.Note{
		Type: "sheet", Title: "Sheet", Body: "# Part\n## Q ^k7m2xq9fzp\n",
		Headings: []domain.Heading{{Level: 1, Text: "Part"}, {Level: 2, Text: "Q ^k7m2xq9fzp"}, {Level: 2, Text: "^zzzz"}, {Level: 3, Text: "Front"}},
	}
	bareNote := domain.Note{Type: "bare", Title: "B", Headings: []domain.Heading{{Level: 1, Text: "Face"}}}
	ordinary := domain.Note{Type: domain.TypeNote, Title: "O", Body: "# One\n", Headings: []domain.Heading{{Level: 1, Text: "One ^kept"}}}

	cutResult := cuts.Execute([]domain.Note{sheet, bareNote, ordinary})

	if len(cutResult[0].Chunks) == 0 {
		t.Errorf("a profile that says nothing of chunks has the note cut: %+v", cutResult[0])
	}
	if got := cutResult[0].Headings; len(got) != 2 || got[0].Text != "Part" || got[1].Text != "Q" {
		t.Errorf("the sheet keeps its section and its card without the mark, got %+v", got)
	}
	if len(cutResult[1].Headings) != 0 {
		t.Errorf("a type with no headings keeps %+v", cutResult[1].Headings)
	}
	if got := cutResult[2].Headings; len(got) != 1 || got[0].Text != "One ^kept" {
		t.Errorf("an ordinary note keeps every heading as it is, got %+v", got)
	}
}

func TestANoteOfATypeWithNoChunksIsNotSplit(t *testing.T) {
	cuts := note.CutNotes{Profiles: func(domain.NoteType) (domain.NoteProfile, bool) {
		return domain.NoteProfile{HasChunks: false}, true
	}}
	deck := domain.Note{Type: "deck", Title: "Deck", Body: "# Part\n"}

	cutResult := cuts.Execute([]domain.Note{deck})

	if len(cutResult[0].Chunks) != 0 {
		t.Errorf("a note whose type has no chunks was cut: %+v", cutResult[0].Chunks)
	}
}
