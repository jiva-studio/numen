package index

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestListPropertyValuesReadsATopLevelKeyByItsExactName(t *testing.T) {
	db := openDB(t)
	saveNoteWithFrontmatter(t, db, getFirstVault(), "a.md", map[string]any{"a.b": 7, "it's": 7.5, "list": []any{"x", 2}}, "Text.")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "b.md", map[string]any{"a": map[string]any{"b": 7}}, "Text.")
	saveNoteWithFrontmatter(t, db, getFirstVault(), "c.md", nil, "Text.")
	queries := db.NoteQueryReads()
	entries, err := queries.ListPropertyValues(t.Context(), getFirstVault().ID, "a.b")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Path != "a.md" || entries[0].Value.Kind != searchquery.PropertyInt || entries[0].Value.Int != 7 {
		t.Errorf("a.b is %+v, want the integer 7 of a.md", entries)
	}
	entries, err = queries.ListPropertyValues(t.Context(), getFirstVault().ID, "it's")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value.Kind != searchquery.PropertyDouble || entries[0].Value.Double != 7.5 {
		t.Errorf("it's is %+v, want the double 7.5", entries)
	}
	entries, err = queries.ListPropertyValues(t.Context(), getFirstVault().ID, "list")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value.Kind != searchquery.PropertyList || len(entries[0].Value.Items) != 2 || entries[0].Value.Items[1].Kind != searchquery.PropertyInt {
		t.Errorf("list is %+v, want a list of a string and an integer", entries)
	}
}

func TestFindHeadingsReturnsASupersetOfTheWholeHeading(t *testing.T) {
	db := openDB(t)
	saveNoteWithHeadings(t, db, getFirstVault(), "a.md", domain.Heading{Level: 2, Text: "summary", Line: 0})
	saveNoteWithHeadings(t, db, getFirstVault(), "b.md", domain.Heading{Level: 2, Text: "Summary of the week", Line: 0})
	saveNoteWithHeadings(t, db, getFirstVault(), "c.md", domain.Heading{Level: 2, Text: "Other", Line: 0})
	found, err := db.NoteQueryReads().FindHeadings(t.Context(), getFirstVault().ID, "Summary")
	if err != nil {
		t.Fatal(err)
	}
	if got := getHeadingPaths(found); !slices.Equal(got, []string{"a.md", "b.md"}) {
		t.Errorf("Summary found %v, want a.md and b.md", got)
	}
	found, err = db.NoteQueryReads().FindHeadings(t.Context(), getFirstVault().ID, "—")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 3 {
		t.Errorf("a text without a word found %+v, want every heading", found)
	}
}

func getHeadingPaths(entries []searchquery.HeadingEntry) []string {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	return paths
}

func TestFindHeadingsReturnsEveryHeadingForATextTheTokenizerSplitsIntoNothing(t *testing.T) {
	db := openDB(t)
	adlam := "\U0001E900\U0001E901"
	saveNoteWithHeadings(t, db, getFirstVault(), "a.md", domain.Heading{Level: 2, Text: adlam, Line: 0})
	saveNoteWithHeadings(t, db, getFirstVault(), "b.md", domain.Heading{Level: 2, Text: "Other", Line: 0})
	found, err := db.NoteQueryReads().FindHeadings(t.Context(), getFirstVault().ID, adlam)
	if err != nil {
		t.Fatal(err)
	}
	if got := getHeadingPaths(found); !slices.Contains(got, "a.md") {
		t.Errorf("the text found %v, want a.md among them", got)
	}
}
