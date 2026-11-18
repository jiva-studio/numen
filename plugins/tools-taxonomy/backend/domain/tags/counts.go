package tags

import (
	"cmp"
	"maps"
	"slices"
)

// Count is a tag with the number of notes that have it.
type Count struct {
	Tag   string
	Count int
}

// GetCounts returns each tag of the notes, by path, once with the number of notes that have exactly that tag, the
// larger count first and then by name. A tag is spelled as the first note in path order spells it.
func GetCounts(notes map[string][]string) []Count {
	byKey := map[string]*Count{}
	for _, path := range slices.Sorted(maps.Keys(notes)) {
		for _, tag := range notes[path] {
			key := GetKey(tag)
			if found, ok := byKey[key]; ok {
				found.Count++
			} else {
				byKey[key] = &Count{Tag: tag, Count: 1}
			}
		}
	}
	out := make([]Count, 0, len(byKey))
	for _, found := range byKey {
		out = append(out, *found)
	}
	slices.SortFunc(out, func(x, y Count) int {
		return cmp.Or(cmp.Compare(y.Count, x.Count), cmp.Compare(GetKey(x.Tag), GetKey(y.Tag)), cmp.Compare(x.Tag, y.Tag))
	})
	return out
}
