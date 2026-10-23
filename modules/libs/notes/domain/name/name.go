package name

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// folder case folds a name. It is stateless, and any number of goroutines may
// use it at once.
var folder = cases.Fold()

// FoldName is the key under which two names are one name: composed, case
// folded, and composed again, since folding `İ` yields `i` and a combining dot.
// Every decision that two names are the same is this key compared.
//
// The fold is the one every reader gets, so `İstanbul` and `istanbul` are two
// names here. The form is NFC: a disk holds `Ｎｏｔｅ` and `Note` as two files,
// and so does this.
func FoldName(name string) string {
	if folded, ok := foldASCII(name); ok {
		return folded
	}
	return foldGeneral(name)
}

// foldGeneral is the fold of any name.
func foldGeneral(name string) string {
	return norm.NFC.String(folder.String(norm.NFC.String(name)))
}

// foldASCII folds a name made only of ASCII, where composing changes nothing and
// folding lower-cases A to Z. ok is false for any other name, and a name already
// folded is returned as it is.
func foldASCII(name string) (folded string, ok bool) {
	hasUpper := false
	for i := 0; i < len(name); i++ {
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
	lower.Grow(len(name))
	for i := 0; i < len(name); i++ {
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

// URLExtension is the extension a web address's file carries.
const URLExtension = ".url"

// LinkName is the name a link is written by: the last segment of what stands
// between the brackets, without a note's extension. The extension comes off
// however it is spelled, because a name is compared without regard to case.
//
// Every other dot belongs to the name: `[[Lecture 1.2]]` names a note filed
// under all of it.
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
