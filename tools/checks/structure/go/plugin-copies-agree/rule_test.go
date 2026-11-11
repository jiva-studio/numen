package plugincopiesagree

import (
	"bytes"
	"fmt"
	"slices"
	"sort"
)

// copySet is a folder that several plugins keep a copy of, because a plugin shares code with no other
// plugin. The copies hold the same files; the files named in Intended may differ.
type copySet struct {
	Folder   string
	Plugins  []string
	Intended []string
}

// listCopySets names the folders the plugins copy from each other.
func listCopySets() []copySet {
	return []copySet{{
		Folder:   "backend/internal/adapter/engine/onnx",
		Plugins:  []string{"engines-embeddings", "engines-ocr", "engines-transcription"},
		Intended: nil,
	}}
}

// getCopyDrifts compares the files of each copy, by name and content, with those of the first
// copy. A file named in intended may differ in content, and must exist in every copy.
func getCopyDrifts(copies map[string]map[string][]byte, order, intended []string) []string {
	var drifts []string
	if len(order) < 2 {
		return []string{fmt.Sprintf("%d copies to compare: a rule checked against nothing", len(order))}
	}
	first := copies[order[0]]
	for _, name := range order[1:] {
		other := copies[name]
		for _, file := range sortedNames(first) {
			content, isThere := other[file]
			switch {
			case !isThere:
				drifts = append(drifts, fmt.Sprintf("%s has no %s, which %s has", name, file, order[0]))
			case !bytes.Equal(content, first[file]) && !slices.Contains(intended, file):
				drifts = append(drifts, fmt.Sprintf("%s differs between %s and %s", file, order[0], name))
			}
		}
		for _, file := range sortedNames(other) {
			if _, isThere := first[file]; !isThere {
				drifts = append(drifts, fmt.Sprintf("%s has %s, which %s has not", name, file, order[0]))
			}
		}
	}
	return drifts
}

func sortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
