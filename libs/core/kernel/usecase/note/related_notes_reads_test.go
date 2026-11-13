package note_test

import (
	"context"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// countingLinks counts how many times each note's links are read.
type countingLinks struct {
	graph
	links     map[string]int
	backlinks map[string]int
}

func (c *countingLinks) GetLinks(ctx context.Context, v domain.VaultID, from string) ([]domain.ResolvedLink, error) {
	c.links[from]++
	return c.graph.GetLinks(ctx, v, from)
}

func (c *countingLinks) GetBacklinks(ctx context.Context, v domain.VaultID, to string) ([]domain.ResolvedLink, error) {
	c.backlinks[to]++
	return c.graph.GetBacklinks(ctx, v, to)
}

func TestEachListOfLinksIsReadOnceAtDepthTwo(t *testing.T) {
	t.Parallel()
	counted := &countingLinks{graph: depthGraph(), links: map[string]int{}, backlinks: map[string]int{}}

	_, err := query.NewListRelatedNotes(counted, &pathLookup{}).Execute(
		context.Background(), domain.Vault{ID: graphVault}, "F.md", 2)
	if err != nil {
		t.Fatal(err)
	}

	for path, n := range counted.links {
		if n != 1 {
			t.Errorf("the links of %s were read %d times, want once", path, n)
		}
	}
	for path, n := range counted.backlinks {
		if n != 1 {
			t.Errorf("the backlinks of %s were read %d times, want once", path, n)
		}
	}
}
