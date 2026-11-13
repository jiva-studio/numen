package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetLinksOf returns GetLinks for each of the paths. A path with no links is absent.
func (q *Queries) GetLinksOf(ctx context.Context, vaultID domain.VaultID, from []string) (map[string][]domain.ResolvedLink, error) {
	return readEach(from, func(path string) ([]domain.ResolvedLink, error) { return q.GetLinks(ctx, vaultID, path) })
}

// GetBacklinksOf returns GetBacklinks for each of the paths. A path with no backlinks is absent.
func (q *Queries) GetBacklinksOf(ctx context.Context, vaultID domain.VaultID, to []string) (map[string][]domain.ResolvedLink, error) {
	return readEach(to, func(path string) ([]domain.ResolvedLink, error) { return q.GetBacklinks(ctx, vaultID, path) })
}

func readEach(paths []string, read func(path string) ([]domain.ResolvedLink, error)) (map[string][]domain.ResolvedLink, error) {
	out := make(map[string][]domain.ResolvedLink, len(paths))
	for _, path := range paths {
		if _, done := out[path]; done {
			continue
		}
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
