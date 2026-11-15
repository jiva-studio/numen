package hosttest

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// The limits of one read of frontmatter, as the host has them.
const (
	frontmatterPageSize = 1000
	frontmatterMaxPaths = 1000
	frontmatterMaxKeys  = 64
)

// ReadFrontmatter returns a page of the frontmatter of the notes in the index, as the host does: in
// path order, with the values JSON gives back.
func (v *Vault) ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return host.FrontmatterPage{}, err
	}
	switch {
	case len(q.Paths) > frontmatterMaxPaths, len(q.Keys) > frontmatterMaxKeys, q.PageSize < 0, q.PageSize > frontmatterPageSize:
		return host.FrontmatterPage{}, host.ErrInvalidFrontmatterQuery
	}
	limit := q.PageSize
	if limit == 0 {
		limit = frontmatterPageSize
	}
	notes := v.parsed()
	paths := sortedPaths(notes)
	if len(q.Paths) > 0 {
		paths = slices.Compact(slices.Sorted(slices.Values(q.Paths)))
	}
	page := host.FrontmatterPage{Cursor: v.feed.getCursor()}
	for _, path := range paths {
		note, ok := notes[path]
		if !ok || path <= q.PageToken {
			continue
		}
		values, err := pickValues(note.Frontmatter, q.Keys)
		if err != nil {
			return host.FrontmatterPage{}, fmt.Errorf("%s: %w", path, err)
		}
		page.Notes = append(page.Notes, host.NoteFrontmatter{SourceID: v.feed.getSourceID(path), Path: path, Values: values})
		if len(page.Notes) == limit {
			page.NextPageToken = path
			break
		}
	}
	return page, nil
}

func pickValues(frontmatter map[string]any, keys []string) (map[string]any, error) {
	picked := map[string]any{}
	for key, value := range frontmatter {
		if len(keys) == 0 || slices.Contains(keys, key) {
			picked[key] = value
		}
	}
	raw, err := json.Marshal(picked)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	return out, json.Unmarshal(raw, &out)
}
