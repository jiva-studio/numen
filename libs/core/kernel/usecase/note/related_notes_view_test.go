package note_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// everyType gives every path it is asked about the type "book".
type everyType struct{}

func (everyType) GetTypes(_ context.Context, _ domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	out := map[string]domain.NoteType{}
	for _, p := range paths {
		out[p] = "book"
	}
	return out, nil
}

// everyTabKind gives every path it is asked about the tab kind "reader".
type everyTabKind struct{}

func (everyTabKind) Execute(_ context.Context, _ domain.Vault, paths []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range paths {
		out[p] = "reader"
	}
	return out, nil
}

func TestATabKindIsChosenForTheFocusAndTheNotesRelatedToIt(t *testing.T) {
	t.Parallel()
	g := depthGraph()
	view := query.NewGetRelatedNotesView(g, &pathLookup{}, everyType{}, everyTabKind{})

	got, err := view.Execute(t.Context(), domain.Vault{ID: graphVault}, "F.md", domain.DepthFar)
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"C.md", "F.md", "P.md", "S.md"}; !slices.Equal(slices.Sorted(maps.Keys(got.TabKinds)), want) {
		t.Errorf("tab kinds of %v, want %v", slices.Sorted(maps.Keys(got.TabKinds)), want)
	}
	for _, r := range got.Related {
		if got.Types[r.Path] != "book" {
			t.Errorf("%s has no type", r.Path)
		}
	}
}
