package catalog

import (
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func mergeEntries(lists [][]Entry) []Entry {
	byID := map[string]Entry{}
	for _, list := range lists {
		for _, entry := range list {
			current, exists := byID[entry.ID]
			if !exists || precedes(entry, current) {
				byID[entry.ID] = entry
			}
		}
	}
	merged := make([]Entry, 0, len(byID))
	for _, entry := range byID {
		merged = append(merged, entry)
	}
	slices.SortFunc(merged, func(a, b Entry) int { return strings.Compare(a.ID, b.ID) })
	return merged
}

func precedes(candidate, current Entry) bool {
	if candidate.State.IsOnDisk() != current.State.IsOnDisk() {
		return candidate.State.IsOnDisk()
	}
	order, err := manifest.CompareVersions(candidate.Version, current.Version)
	return err == nil && order > 0
}
