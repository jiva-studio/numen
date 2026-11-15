// Package tags reads the tags of a note from the value of its frontmatter key tags and counts the
// notes of each tag. A tag nests under another with "/", and two spellings that differ only in case or
// in Unicode composition are one tag.
package tags

import (
	"cmp"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// FrontmatterKey is the key of the frontmatter that holds the tags of a note.
const FrontmatterKey = "tags"

// Parse returns the tags of the value of the key tags: one for a string or a number, one for each
// string or number of a list, none for anything else. Each tag is cleaned as Clean does, and a tag
// that cleans to nothing or repeats another is left out.
func Parse(value any) []string {
	var raw []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			if text, ok := readText(item); ok {
				raw = append(raw, text)
			}
		}
	default:
		if text, ok := readText(v); ok {
			raw = append(raw, text)
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, one := range raw {
		tag := Clean(one)
		key := cases.Fold().String(tag)
		if tag == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}

func readText(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case json.Number:
		return v.String(), true
	case int:
		return strconv.Itoa(v), true
	}
	return "", false
}

// Clean returns the tag as it is kept: without the spaces around it and one leading "#", in Unicode
// NFC, with each segment between "/" trimmed and the empty segments dropped.
func Clean(tag string) string {
	tag = norm.NFC.String(strings.TrimPrefix(strings.TrimSpace(tag), "#"))
	var segments []string
	for segment := range strings.SplitSeq(tag, "/") {
		if segment = strings.TrimSpace(segment); segment != "" {
			segments = append(segments, segment)
		}
	}
	return strings.Join(segments, "/")
}

// GetKey returns what two spellings of one tag share: the cleaned tag with its case folded.
func GetKey(tag string) string {
	return cases.Fold().String(Clean(tag))
}

// Node is a tag in the tree of tags: the last segment as first spelled, the whole tag, the number of
// notes that have it or a tag nested under it, and the tags nested one level under it.
type Node struct {
	Name     string
	Tag      string
	Count    int
	Children []Node
}

// GetTree returns the tree of the tags of the notes, by path. A tag is spelled as the first note in
// path order spells it. Siblings come by count, the larger first, then by name. With under, the tree
// holds that tag alone with what is nested under it, or nothing when no note has it.
func GetTree(notes map[string][]string, under string) []Node {
	root := &branch{name: "", tag: "", notes: map[string]bool{}, children: map[string]*branch{}}
	for _, path := range slices.Sorted(maps.Keys(notes)) {
		for _, tag := range notes[path] {
			root.add(path, tag)
		}
	}
	if strings.TrimSpace(under) == "" {
		return root.listNodes()
	}
	node := root
	for segment := range strings.SplitSeq(GetKey(under), "/") {
		next, ok := node.children[segment]
		if !ok {
			return []Node{}
		}
		node = next
	}
	return []Node{node.getNode()}
}

// branch is a tag of the tree being built, with the notes under it and the tags nested under it by
// the folded segment.
type branch struct {
	name     string
	tag      string
	notes    map[string]bool
	children map[string]*branch
}

func (b *branch) add(path, tag string) {
	node := b
	for segment := range strings.SplitSeq(tag, "/") {
		key := cases.Fold().String(segment)
		next, ok := node.children[key]
		if !ok {
			next = &branch{name: segment, tag: segment, notes: map[string]bool{}, children: map[string]*branch{}}
			if node.tag != "" {
				next.tag = node.tag + "/" + segment
			}
			node.children[key] = next
		}
		next.notes[path] = true
		node = next
	}
}

func (b *branch) getNode() Node {
	return Node{Name: b.name, Tag: b.tag, Count: len(b.notes), Children: b.listNodes()}
}

func (b *branch) listNodes() []Node {
	out := make([]Node, 0, len(b.children))
	for _, child := range b.children {
		out = append(out, child.getNode())
	}
	slices.SortFunc(out, func(x, y Node) int {
		return cmp.Or(cmp.Compare(y.Count, x.Count), cmp.Compare(GetKey(x.Name), GetKey(y.Name)), cmp.Compare(x.Name, y.Name))
	})
	return out
}
