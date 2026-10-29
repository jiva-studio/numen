package note_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
)

func TestGetTitles_PreservesRequestOrderAndReportsMissing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n", "Work.md": "# Work\n"})

	got, err := note.NewGetTitles(c.db.Queries()).Execute(t.Context(), c.vault,
		[]string{"Work.md", "Gone.md", "Heat.md", "Lost.md"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, ref := range got.Notes {
		paths = append(paths, ref.Path)
	}
	if !slices.Equal(paths, []string{"Work.md", "Heat.md"}) {
		t.Errorf("the notes came back as %v", paths)
	}
	if !slices.Equal(got.Missing, []string{"Gone.md", "Lost.md"}) {
		t.Errorf("the missing came back as %v", got.Missing)
	}
}
