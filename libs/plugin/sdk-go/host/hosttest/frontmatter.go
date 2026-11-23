package hosttest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	notesmarkdown "github.com/jiva-studio/numen/libs/shared/note/markdown"
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
	if err := v.checkContext(ctx); err != nil {
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
	paths := listSortedPaths(notes)
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

// WriteFrontmatter sets, replaces and removes top-level keys of the notes, as the host does: a list of
// text is written as a list, a nil value removes the key, a stale note fails alone with host.ErrStale.
func (v *Vault) WriteFrontmatter(ctx context.Context, writes []host.FrontmatterWrite) ([]host.FrontmatterWritten, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	out := make([]host.FrontmatterWritten, len(writes))
	for i, one := range writes {
		stat, err := v.editLinks(ctx, one.Path, one.Baseline, func(doc *notesmarkdown.Document) error {
			return setKeys(doc, one.Values)
		})
		out[i] = host.FrontmatterWritten{Stat: stat, Err: err}
	}
	return out, nil
}

func setKeys(doc *notesmarkdown.Document, values map[string]any) error {
	for _, key := range slices.Sorted(maps.Keys(values)) {
		var err error
		switch value := values[key].(type) {
		case nil:
			err = doc.Remove(key)
		case []string:
			err = doc.SetList(key, value)
		case []any:
			if names, isText := convertNames(value); isText {
				err = doc.SetList(key, names)
			} else {
				err = doc.SetValue(key, value)
			}
		default:
			err = doc.SetValue(key, value)
		}
		if err != nil {
			return fmt.Errorf("write %q: %w", key, err)
		}
	}
	return nil
}

func convertNames(items []any) ([]string, bool) {
	names := make([]string, len(items))
	for i, item := range items {
		name, isText := item.(string)
		if !isText {
			return nil, false
		}
		names[i] = name
	}
	return names, true
}
