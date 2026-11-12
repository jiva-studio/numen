package domain

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// FoldName is the key under which two names are one name: composed, case
// folded, and composed again, since folding `İ` yields `i` and a combining dot.
// The fold is locale-independent, so `İstanbul` and `istanbul` are two names.
// The form is NFC, but `Ｎｏｔｅ` and `Note` stay two names, as on disk.
func FoldName(name string) string {
	if folded, ok := foldASCII(name); ok {
		return folded
	}
	return foldGeneral(name)
}

func foldGeneral(name string) string {
	return norm.NFC.String(cases.Fold().String(norm.NFC.String(name)))
}

func foldASCII(name string) (folded string, ok bool) {
	hasUpper := false
	for i := range len(name) {
		c := name[i]
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="the mutated bound differs only for the byte 0x80, which the general fold leaves as it is, so both paths return the same name"
		if c >= utf8.RuneSelf {
			return "", false
		}
		if 'A' <= c && c <= 'Z' {
			hasUpper = true
		}
	}
	if !hasUpper {
		return name, true
	}
	var lower strings.Builder
	// gomutants:disable-next-line STATEMENT_REMOVE reason="strings.Builder.Grow is a capacity hint that does not alter the built string"
	lower.Grow(len(name))
	for i := range len(name) {
		c := name[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		lower.WriteByte(c)
	}
	return lower.String(), true
}

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

// LinkName is the name a link is written by: the last segment of the text
// between the brackets, without a note's extension.
// The extension is stripped however it is spelled; every other dot belongs to
// the name, so `[[Lecture 1.2]]` names a note filed under all of it.
func LinkName(target string) string {
	name := target
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	if baseLen := len(name) - len(NoteExtension); baseLen > 0 &&
		FoldName(name[baseLen:]) == NoteExtension {
		return name[:baseLen]
	}
	return name
}

// HasPathPrefix reports whether the composed (NFC) form of the path starts with the composed form of
// the prefix.
func HasPathPrefix(path, prefix string) bool {
	return strings.HasPrefix(norm.NFC.String(path), norm.NFC.String(prefix))
}
