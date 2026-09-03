package domain

import "strings"

// Basename is the name a note is filed under: the last segment of its path,
// without its extension.
//
// The scan stores what this returns, and it is what a name is matched against.
func Basename(path string) string {
	name := path
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	// A leading dot is part of the name, not the start of an extension.
	if i := strings.LastIndexByte(name, '.'); i > 0 {
		name = name[:i]
	}
	return name
}

// NoteExtension is the extension a note's file carries.
const NoteExtension = ".md"

// LinkName is the name a link is written by: the last segment of what stands
// between the brackets, without a note's extension.
//
// Every other dot belongs to the name: `[[Lecture 1.2]]` names a note filed
// under all of it.
func LinkName(written string) string {
	name := written
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	if trimmed := strings.TrimSuffix(name, NoteExtension); trimmed != "" {
		return trimmed
	}
	return name
}
