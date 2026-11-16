package tags

import "slices"

// Apply returns the tags of a note after the toggles: a toggled tag the note has is removed, and one it has not is
// appended, in the order of the toggles. Tags are compared as GetKey compares them, the tags the note keeps stay in its
// order and in their spelling, and each toggle is cleaned as Clean does.
func Apply(current []string, toggles []string) []string {
	out := slices.Clone(current)
	for _, toggle := range toggles {
		clean := Clean(toggle)
		if clean == "" {
			continue
		}
		key := GetKey(clean)
		if at := slices.IndexFunc(out, func(tag string) bool { return GetKey(tag) == key }); at >= 0 {
			out = slices.Delete(out, at, at+1)
		} else {
			out = append(out, clean)
		}
	}
	return out
}
