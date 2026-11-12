package chunking

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestNoteRecipeDiffersBetweenChunkSizes(t *testing.T) {
	small, large := NewCutNotes(256).Recipe(), NewCutNotes(512).Recipe()
	if small == large {
		t.Errorf("a cut at 256 tokens and one at 512 tokens share the recipe %q", small)
	}
}

func TestNoteRecipeIsTheSameForEqualChunkSizes(t *testing.T) {
	first, second := NewCutNotes(512).Recipe(), NewCutNotes(512).Recipe()
	if first == "" || first != second {
		t.Errorf("two cuts at 512 tokens have the recipes %q and %q, want one non-empty recipe", first, second)
	}
}

func TestNoteRecipeNamesTheCharacterLimit(t *testing.T) {
	if got := NewCutNotes(512).Recipe(); !strings.Contains(got, "limit=1024") {
		t.Errorf("recipe %q of a cut at 512 tokens does not name the limit of 1024 characters", got)
	}
}

func TestNoteRecipeWithNoSizeNamesTheDefaultLimit(t *testing.T) {
	if got := NewCutNotes(0).Recipe(); !strings.Contains(got, "limit=1000") {
		t.Errorf("recipe %q of a cut with no size does not name the default limit of 1000 characters", got)
	}
}

func TestNoteRecipeIgnoresLegibility(t *testing.T) {
	strict := NewCutNotes(512)
	strict.Legibility = Legibility{Alphabetic: 0.9}
	if strict.Recipe() == "" || strict.Recipe() != NewCutNotes(512).Recipe() {
		t.Errorf("a stricter legibility changed the recipe: %q", strict.Recipe())
	}
}

func TestExecuteStampsEveryNoteWithTheRecipe(t *testing.T) {
	notes := []domain.Note{
		{Title: "Hedge", Body: line(40), FileStat: domain.FileStat{Size: int64(len(line(40)))}},
		{Title: "Gate", Body: line(5), FileStat: domain.FileStat{Size: int64(len(line(5)))}},
	}
	for _, tokens := range []int{256, 512} {
		splitter := NewCutNotes(tokens)
		for _, got := range splitter.Execute(notes) {
			if got.Recipe == "" || got.Recipe != splitter.Recipe() {
				t.Errorf("note %q cut at %d tokens has the recipe %q, want %q", got.Note.Title, tokens, got.Recipe, splitter.Recipe())
			}
		}
	}
}
