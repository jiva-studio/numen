package domain

import (
	"path"
	"slices"
	"strings"
)

// ChooseMatch applies the priority a name resolves by: an exact path from the root,
// then a path relative to the note the link is written in, then a single match by name.
// Several matches are ambiguous: the link resolves to the nearest one in the tree.
// A path is matched under FoldName.
func ChooseMatch(from, name string, candidates []string) (bestMatch string, ambiguous bool) {
	if len(candidates) == 0 {
		return "", false
	}

	folded := make([]string, len(candidates))
	for i, c := range candidates {
		folded[i] = FoldName(c)
	}

	// A name is written with an extension or without one, and both spellings
	// name the same file.
	wanted := []string{name, name + NoteExtension}
	for _, w := range wanted {
		if idx := slices.Index(folded, FoldName(w)); idx >= 0 {
			return candidates[idx], false
		}
	}

	for _, w := range wanted {
		relative := FoldName(path.Join(path.Dir(from), w))
		if idx := slices.Index(folded, relative); idx >= 0 {
			return candidates[idx], false
		}
	}

	if len(candidates) == 1 {
		return candidates[0], false
	}

	// Nearest by how much of the path they have in common with the note the
	// link was written in — the neighbor is likelier than the stranger.
	var best string
	var bestShared int
	for _, c := range candidates {
		if shared := countSharedPrefix(from, c); best == "" || shared > bestShared {
			best, bestShared = c, shared
		}
	}
	return best, true
}

func countSharedPrefix(a, b string) int {
	as, bs := strings.Split(path.Dir(a), "/"), strings.Split(path.Dir(b), "/")
	n := 0
	for n < len(as) && n < len(bs) && as[n] == bs[n] {
		n++
	}
	return n
}
