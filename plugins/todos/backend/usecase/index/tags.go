package index

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

const (
	tagsKey       = "tags"
	readPathLimit = 1000
)

func (i *Index) refreshTags(ctx context.Context, paths []string) error {
	changed := false
	for part := range slices.Chunk(slices.Compact(slices.Sorted(slices.Values(paths))), readPathLimit) {
		found, err := i.readTags(ctx, part)
		if err != nil {
			return err
		}
		changed = i.replaceTags(part, found) || changed
	}
	if changed {
		i.mu.Lock()
		i.bump()
		i.mu.Unlock()
	}
	return nil
}

func (i *Index) refreshAllTags(ctx context.Context) error {
	found, err := i.readTags(ctx, nil)
	if err != nil {
		return err
	}
	i.mu.Lock()
	paths := make([]string, 0, len(i.entries))
	for path := range i.entries {
		paths = append(paths, path)
	}
	i.mu.Unlock()
	if i.replaceTags(paths, found) {
		i.mu.Lock()
		i.bump()
		i.mu.Unlock()
	}
	return nil
}

func (i *Index) readTags(ctx context.Context, paths []string) (map[string][]string, error) {
	found := map[string][]string{}
	query := host.FrontmatterQuery{Paths: paths, Keys: []string{tagsKey}}
	for {
		page, err := i.front.ReadFrontmatter(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, note := range page.Notes {
			if tags := parseTags(note.Values[tagsKey]); len(tags) > 0 {
				found[note.Path] = tags
			}
		}
		if page.NextPageToken == "" {
			return found, nil
		}
		query.PageToken = page.NextPageToken
	}
}

func (i *Index) replaceTags(paths []string, found map[string][]string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	isChanged := false
	for _, path := range paths {
		known, isKnown := i.entries[path]
		if !isKnown || slices.Equal(known.tags, found[path]) {
			continue
		}
		i.countTags(known.tags, -1)
		known.tags = found[path]
		i.countTags(known.tags, 1)
		i.entries[path] = known
		isChanged = true
	}
	return isChanged
}

func (i *Index) countTags(tags []string, step int) {
	for _, tag := range tags {
		key := strings.ToLower(tag)
		i.tags[key] += step
		if i.tags[key] <= 0 {
			delete(i.tags, key)
			delete(i.spelling, key)
		} else if spelled, isSpelled := i.spelling[key]; !isSpelled || tag < spelled {
			i.spelling[key] = tag
		}
	}
}

func parseTags(value any) []string {
	var raw []string
	if items, isList := value.([]any); isList {
		for _, item := range items {
			if text, isText := readTagText(item); isText {
				raw = append(raw, text)
			}
		}
	} else if text, isText := readTagText(value); isText {
		raw = append(raw, text)
	}
	var tags []string
	seen := map[string]bool{}
	for _, one := range raw {
		tag := cleanTag(one)
		key := strings.ToLower(tag)
		if tag != "" && !seen[key] {
			seen[key] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

func cleanTag(tag string) string {
	var segments []string
	for segment := range strings.SplitSeq(strings.TrimPrefix(strings.TrimSpace(tag), "#"), "/") {
		if segment = strings.TrimSpace(segment); segment != "" {
			segments = append(segments, segment)
		}
	}
	return strings.Join(segments, "/")
}

func readTagText(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return strconv.Itoa(v), true
	}
	return "", false
}
