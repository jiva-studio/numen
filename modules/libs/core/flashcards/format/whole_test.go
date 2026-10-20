package format_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func wholeAnimals() map[string]format.Stencil {
	return map[string]format.Stencil{"Animal": {Fields: []string{"Name"}}}
}

func createWholeID() domain.CardID { return "zpqrstvwxy" }

// A heading that has fallen out of step with the card's first field is written
// again from the first line of that field, with the mark the card carries.
func TestWholeWritesTheHeadingFromTheFirstField(t *testing.T) {
	body := "\n## Old ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\nthe woolly one\n"

	got, given := format.Whole(body, wholeAnimals(), createWholeID)

	want := "\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\nthe woolly one\n"
	if got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
	if len(given) != 0 {
		t.Errorf("given = %+v, want no mark given", given)
	}
}

// A card of a stencil nobody holds keeps the heading it has, and is given its
// mark all the same.
func TestWholeKeepsTheHeadingOfACardWhoseStencilIsUnknown(t *testing.T) {
	body := "\n## Old\n\n[[Plant]]\n\n### Name\n\nFern\n"

	got, _ := format.Whole(body, wholeAnimals(), createWholeID)

	want := "\n## Old ^zpqrstvwxy\n\n[[Plant]]\n\n### Name\n\nFern\n"
	if got != want {
		t.Errorf("body\n want %q\n  got %q", want, got)
	}
}

// A heading already in step is not written again: the spacing around it stays
// as the person typed it.
func TestWholeLeavesAHeadingInStepAsItStands(t *testing.T) {
	body := "\n##   Llama ^k7m2xq9fzp  \n\n[[Animal]]\n\n### Name\n\nLlama\n"

	got, given := format.Whole(body, wholeAnimals(), createWholeID)

	if got != body {
		t.Errorf("body\n want %q\n  got %q", body, got)
	}
	if len(given) != 0 {
		t.Errorf("given = %+v", given)
	}
}

// A heading that ends the body is written without a line break after it.
func TestWholeWritesTheLastHeadingWithoutAnEnding(t *testing.T) {
	got, given := format.Whole("## Llama", nil, createWholeID)

	if want := "## Llama ^zpqrstvwxy"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if len(given) != 1 {
		t.Errorf("given = %+v", given)
	}
}
