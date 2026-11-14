package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetLinksOf runs one GetLinks query per path; the batch is a loop, not one SQL statement.
func (q *Queries) GetLinksOf(ctx context.Context, vaultID domain.VaultID, from []string) (map[string][]domain.ResolvedLink, error) {
	return readEach(ctx, from, func(path string) ([]domain.ResolvedLink, error) { return q.GetLinks(ctx, vaultID, path) })
}

// GetBacklinksOf runs one GetBacklinks query per path; the batch is a loop, not one SQL statement.
func (q *Queries) GetBacklinksOf(ctx context.Context, vaultID domain.VaultID, to []string) (map[string][]domain.ResolvedLink, error) {
	return readEach(ctx, to, func(path string) ([]domain.ResolvedLink, error) { return q.GetBacklinks(ctx, vaultID, path) })
}

func readEach(ctx context.Context, paths []string, read func(path string) ([]domain.ResolvedLink, error)) (map[string][]domain.ResolvedLink, error) {
	out := make(map[string][]domain.ResolvedLink, len(paths))
	visited := map[string]bool{}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if visited[path] {
			continue
		}
		visited[path] = true
		links, err := read(path)
		if err != nil {
			return nil, err
		}
		if len(links) > 0 {
			out[path] = links
		}
	}
	return out, nil
}
