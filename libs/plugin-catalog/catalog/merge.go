package catalog

import (
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// mergeEntries returns one entry for each plugin id, ordered by id. Of the entries sharing an id, the
// one on disk wins over the one on offer; between two of the same standing the newer version wins;
// and between two of the same version the one from the earlier list wins.
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

// precedes reports whether the candidate replaces the entry current for its id.
func precedes(candidate, current Entry) bool {
	if candidate.State.IsOnDisk() != current.State.IsOnDisk() {
		return candidate.State.IsOnDisk()
	}
	order, err := manifest.CompareVersions(candidate.Version, current.Version)
	return err == nil && order > 0
}
