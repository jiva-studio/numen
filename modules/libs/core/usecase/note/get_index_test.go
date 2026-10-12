package note_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func TestGetOpeningNoteAnswersTheNoteTheVaultOpensOn(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetOpeningNote(c.db.Queries()).Execute(t.Context(), c.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Path != "Heat.md" {
		t.Errorf("the vault opens on %v", got)
	}
}

func TestGetOpeningNoteAnswersNothingForAVaultHoldingNoNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{})

	got, err := note.NewGetOpeningNote(c.db.Queries()).Execute(t.Context(), c.vault)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("the vault opens on %v", got)
	}
}

func TestGetTitleAnswersTheTitleOfTheNoteAtAPath(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	title := note.NewGetTitle(c.db.Queries())

	if got := title.Execute(t.Context(), c.vault, "Heat.md"); got == "" {
		t.Error("a note in the index has no title")
	}
	if got := title.Execute(t.Context(), c.vault, "Gone.md"); got != "" {
		t.Errorf("a path naming no note is titled %q", got)
	}
	if got := title.Execute(t.Context(), c.vault, ""); got != "" {
		t.Errorf("no path is titled %q", got)
	}
}

func TestGetTypesAnswersByPathAndLeavesOutWhatIsNoNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetTypes(c.db.Queries()).Execute(t.Context(), c.vault, []string{"Heat.md", "Gone.md"})
	if err != nil {
		t.Fatal(err)
	}
	if _, held := got["Heat.md"]; !held {
		t.Errorf("the types are %v", got)
	}
	if _, held := got["Gone.md"]; held {
		t.Errorf("a path naming no note has the type %v", got["Gone.md"])
	}
}

func TestGetNameMatchesAnswersTheNotesWhoseNamesMatch(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n", "Work.md": "# Work\n"})

	got, err := note.NewGetNameMatches(c.db.Queries()).Execute(t.Context(), c.vault, "Heat", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[0].Path != "Heat.md" {
		t.Errorf("the matches are %v", got)
	}
}

func TestGetHeadingsAnswersByPath(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n\n## Flow\n", "Work.md": "Work\n"})

	got, err := note.NewGetHeadings(c.db.Queries()).Execute(t.Context(), c.vault, []string{"Heat.md", "Work.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["Heat.md"]) == 0 {
		t.Errorf("the headings are %v", got)
	}
	if _, held := got["Work.md"]; held {
		t.Errorf("a note with no headings has %v", got["Work.md"])
	}
}

func TestGetResolvedAddressesAnswersWhereAWrittenAddressLands(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n", "Work.md": "# Work\n"})

	got, err := note.NewGetResolvedAddresses(c.db.Links()).Execute(
		t.Context(), c.vault, "Work.md", []string{"Heat", "Nowhere"})
	if err != nil {
		t.Fatal(err)
	}
	if got["Heat"].To != "Heat.md" {
		t.Errorf("Heat lands at %v", got["Heat"])
	}
	if _, held := got["Nowhere"]; held {
		t.Errorf("an address that reaches nothing lands at %v", got["Nowhere"])
	}
}

func TestGetLinkAddressWritesTheNameOfTheNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetLinkAddress(c.db.Queries()).Execute(t.Context(), c.vault, "Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != (domain.Address{Scheme: domain.SchemeName, Value: "Heat"}) {
		t.Errorf("the address is %v", got)
	}
}

func TestGetLinkAddressWritesTheNameWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	got, err := note.NewGetLinkAddress(nil).Execute(t.Context(), c.vault, "physics/Heat.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != (domain.Address{Scheme: domain.SchemeName, Value: "Heat"}) {
		t.Errorf("the address is %v", got)
	}
}
