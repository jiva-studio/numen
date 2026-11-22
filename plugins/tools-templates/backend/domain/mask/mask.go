// Package mask reads the glob masks that say which notes of a vault are templates.
package mask

import (
	pathpkg "path"
	"strings"
)

const (
	anyFolders = "**"
	wildcards  = "*?["
)

func getDefaults() []string { return []string{"Templates/**", "**/*.template.md"} }

// Mask is a glob over vault paths. A segment is matched as path.Match does, and a segment ** stands for any number of folders.
type Mask struct {
	segments []string
}

// Set is the masks of the templates setting, in order.
type Set []Mask

// Parse reads the masks of the setting: entries separated by commas or lines. An entry that climbs out of the vault is dropped,
// and no entry left gives the defaults.
func Parse(raw string) Set {
	var set Set
	entries := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' })
	for _, entry := range entries {
		if one, ok := parseEntry(entry); ok {
			set = append(set, one)
		}
	}
	if len(set) > 0 {
		return set
	}
	for _, entry := range getDefaults() {
		one, _ := parseEntry(entry)
		set = append(set, one)
	}
	return set
}

func parseEntry(entry string) (Mask, bool) {
	var segments []string
	for segment := range strings.SplitSeq(strings.ReplaceAll(strings.TrimSpace(entry), `\`, "/"), "/") {
		switch segment {
		case "", ".":
		case "..":
			return Mask{segments: nil}, false
		default:
			segments = append(segments, segment)
		}
	}
	return Mask{segments: segments}, len(segments) > 0
}

// Match reports whether a vault path is named by the mask.
func (m Mask) Match(path string) bool {
	return matchSegments(m.segments, strings.Split(path, "/"))
}

func matchSegments(pattern, parts []string) bool {
	if len(pattern) == 0 {
		return len(parts) == 0
	}
	if pattern[0] == anyFolders {
		return matchAnyFolders(pattern[1:], parts)
	}
	if len(parts) == 0 {
		return false
	}
	if ok, err := pathpkg.Match(pattern[0], parts[0]); err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], parts[1:])
}

func matchAnyFolders(rest, parts []string) bool {
	for skipped := 0; skipped <= len(parts); skipped++ {
		if matchSegments(rest, parts[skipped:]) {
			return true
		}
	}
	return false
}

func (m Mask) getStaticFolder() []string {
	var kept []string
	for _, segment := range m.segments[:len(m.segments)-1] {
		if strings.ContainsAny(segment, wildcards) {
			break
		}
		kept = append(kept, segment)
	}
	return kept
}

// Group is the folder of a path the mask names, below the static folder the mask starts with, and the whole folder where it has none.
func (m Mask) Group(path string) string {
	folder := strings.Split(pathpkg.Dir(path), "/")
	if pathpkg.Dir(path) == "." {
		return ""
	}
	prefix := m.getStaticFolder()
	if len(prefix) > len(folder) {
		return ""
	}
	return strings.Join(folder[len(prefix):], "/")
}

// Find returns the group of a path under the first mask that names it, and whether any does.
func (s Set) Find(path string) (string, bool) {
	for _, one := range s {
		if one.Match(path) {
			return one.Group(path), true
		}
	}
	return "", false
}
