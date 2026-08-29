package domain

import "strings"

// Basename is the name a note is found by when a link is written by name: the
// last segment of a path, without its extension.
//
// One function, because the scan stores what it returns and resolution asks for
// it. Two spellings of the same rule differ on `.hidden.md` — one storing
// `.hidden` and the other asking for nothing — and the link then matches
// nothing for a reason no one can see.
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
