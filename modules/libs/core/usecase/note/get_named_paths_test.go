package note_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

func TestGetNamedPathsAnswersEveryNoteFiledBelowAName(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"a/Heat.md": "# Heat\n", "b/Heat.md": "# Heat\n", "Cold.md": "# Cold\n"})
	u := note.NewGetNamedPaths(c.db.Queries())

	got, err := u.Execute(t.Context(), c.vault, "Heat")
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	if want := []string{"a/Heat.md", "b/Heat.md"}; !slices.Equal(got, want) {
		t.Errorf("Heat reaches %v, want %v", got, want)
	}
	if got, _ := u.Execute(t.Context(), c.vault, "Gone"); len(got) != 0 {
		t.Errorf("a name no note answers to reaches %v", got)
	}
	if got, _ := u.Execute(t.Context(), c.vault, ""); len(got) != 0 {
		t.Errorf("no name reaches %v", got)
	}
}

func TestGetNamedPathsAnswersNothingWhereNoIndexIsBound(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

	if got, err := note.NewGetNamedPaths(nil).Execute(t.Context(), c.vault, "Heat"); err != nil || len(got) != 0 {
		t.Errorf("an unbound index answered %v, %v", got, err)
	}
	var unmade *note.GetNamedPaths
	if got, err := unmade.Execute(t.Context(), c.vault, "Heat"); err != nil || len(got) != 0 {
		t.Errorf("a use case not made answered %v, %v", got, err)
	}
}
