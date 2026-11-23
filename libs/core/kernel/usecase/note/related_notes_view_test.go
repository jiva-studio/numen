package note_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
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

func (everyTabKind) ListTabKinds(_ context.Context, _ domain.Vault, paths []string) (map[string]string, error) {
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

	views, err := view.Execute(t.Context(), domain.Vault{ID: graphVault}, []string{"F.md"}, domain.DepthFar)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 {
		t.Fatalf("%d views for one path, want 1", len(views))
	}
	got := views[0]

	if want := []string{"C.md", "F.md", "P.md", "S.md"}; !slices.Equal(slices.Sorted(maps.Keys(got.TabKinds)), want) {
		t.Errorf("tab kinds of %v, want %v", slices.Sorted(maps.Keys(got.TabKinds)), want)
	}
	for _, r := range got.Related {
		if got.Types[r.Path] != "book" {
			t.Errorf("%s has no type", r.Path)
		}
	}
}

// countingTypes records the paths of every question about types.
type countingTypes struct {
	everyType
	calls [][]string
}

func (c *countingTypes) GetTypes(ctx context.Context, v domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	c.calls = append(c.calls, slices.Clone(paths))
	return c.everyType.GetTypes(ctx, v, paths)
}

// countingTabKinds records the paths of every question about tab kinds.
type countingTabKinds struct {
	everyTabKind
	calls [][]string
}

func (c *countingTabKinds) ListTabKinds(ctx context.Context, v domain.Vault, paths []string) (map[string]string, error) {
	c.calls = append(c.calls, slices.Clone(paths))
	return c.everyTabKind.ListTabKinds(ctx, v, paths)
}

func TestTheTypesAndTheTabKindsOfAllTheGraphsAreReadOnce(t *testing.T) {
	t.Parallel()
	types, tabKinds := &countingTypes{}, &countingTabKinds{}
	view := query.NewGetRelatedNotesView(depthGraph(), &pathLookup{}, types, tabKinds)
	paths := []string{"F.md", "S.md"}

	got, err := view.Execute(t.Context(), domain.Vault{ID: graphVault}, paths, domain.DepthNear)
	if err != nil {
		t.Fatal(err)
	}

	if len(types.calls) != 1 || len(tabKinds.calls) != 1 {
		t.Errorf("%d questions about types and %d about tab kinds, want one of each", len(types.calls), len(tabKinds.calls))
	}
	if len(got) != len(paths) {
		t.Fatalf("%d views for %d paths", len(got), len(paths))
	}
	for i, path := range paths {
		if got[i].Focus.Path != path || got[i].Types[path] != "book" || got[i].TabKinds[path] != "reader" {
			t.Errorf("view %d is %+v, want the view of %s with its type and tab kind", i, got[i], path)
		}
	}
}
