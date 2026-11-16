package settags

import (
	"slices"

	"github.com/jiva-studio/numen/plugins/tools-taxonomy/backend/domain/tags"
)

// toggle is one tag of a call, cleaned: it is cleared from every note when every note has it, and set on the notes
// without it otherwise.
type toggle struct {
	tag     string
	key     string
	isClear bool
}

// decideToggles cleans the tags, drops those that clean to nothing and repeats, and decides for each whether it is
// cleared or set.
func decideToggles(paths []string, current map[string][]string, requested []string) []toggle {
	var decided []toggle
	for _, raw := range requested {
		tag := tags.Clean(raw)
		key := tags.GetKey(tag)
		if tag == "" || slices.ContainsFunc(decided, func(one toggle) bool { return one.key == key }) {
			continue
		}
		isClear := len(paths) > 0 && !slices.ContainsFunc(paths, func(path string) bool { return !hasTag(current[path], key) })
		decided = append(decided, toggle{tag: tag, key: key, isClear: isClear})
	}
	return decided
}

// pickToggles returns the toggles that change a note with the tags: a cleared tag it has, and a set tag it lacks.
func pickToggles(decided []toggle, noteTags []string) []toggle {
	var picked []toggle
	for _, one := range decided {
		if one.isClear == hasTag(noteTags, one.key) {
			picked = append(picked, one)
		}
	}
	return picked
}

func getTags(toggles []toggle) []string {
	out := make([]string, len(toggles))
	for i, one := range toggles {
		out[i] = one.tag
	}
	return out
}

func hasTag(noteTags []string, key string) bool {
	return slices.ContainsFunc(noteTags, func(tag string) bool { return tags.GetKey(tag) == key })
}
