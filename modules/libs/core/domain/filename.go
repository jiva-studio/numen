package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrUnnameable is a title no note can be given. Nothing is written.
var ErrUnnameable = errors.New("a note cannot be given this title")

// Filename refuses a title no note can be given, and answers for every other
// with the name it is filed under and whether it survived the trip.
//
// Two titles are refused whatever the note: one that leaves no filename, and
// one carrying a line break. The title comes in trimmed.
func Filename(title string) (name string, exact bool, err error) {
	switch name, exact = GetReducedFilename(title); {
	case name == "":
		return "", false, fmt.Errorf("%w: %q leaves nothing a file can be named after", ErrUnnameable, title)
	case strings.ContainsAny(title, "\n\r"):
		return "", false, fmt.Errorf("%w: %q is more than one line", ErrUnnameable, title)
	}
	return name, exact, nil
}

// maxFilename is how long a name is allowed to get. Most filesystems stop at
// 255 bytes for one component, and a title is not the place to find that out.
const maxFilename = 120

// GetReducedFilename is the filename a title reduces to, refusing nothing: a
// title that reduces to no name at all comes back empty.
//
// A note is shown by its `title`, else by its filename. So a file named after
// the title needs no `title` key at all. When the title cannot be a filename,
// `exact` is false and the caller writes the title into that key instead.
//
// Every name that comes back is one IsNameable accepts, so a link written by it
// reaches the note back.
func GetReducedFilename(title string) (name string, exact bool) {
	var b strings.Builder
	var last rune
	for _, r := range strings.TrimSpace(title) {
		switch {
		case unicode.IsControl(r):
			continue
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' || r == '#':
			// Reserved somewhere that matters. A slash names a folder, and a
			// `#` or a `|` is punctuation of the link a name is written as.
			r = '-'
		case (r == '[' || r == ']') && r == last:
			// A doubled bracket opens or closes a link, so a run of them is one.
			continue
		}
		b.WriteRune(r)
		last = r
	}

	name = trimEnds(b.String())
	if len(name) > maxFilename {
		name = trimEnds(cutRunes(name, maxFilename))
	}
	if name == "" {
		return "", false
	}
	if isDevice(name) {
		return name + "-", false
	}
	return name, name == strings.TrimSpace(title)
}

// isDevice reports whether Windows keeps this name for a device, which no file
// there may carry. The name is the device whatever extension follows it, and
// whatever case it is written in: `con`, `CON.md` and `Con.notes.md` are all
// the console.
func isDevice(name string) bool {
	stem, _, _ := strings.Cut(name, ".")
	stem = strings.ToUpper(stem)
	switch stem {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(stem) != 4 || stem[3] < '1' || stem[3] > '9' {
		return false
	}
	return stem[:3] == "COM" || stem[:3] == "LPT"
}

// trimEnds is a name carrying at neither end a dot or a space. A leading dot
// files the note where nothing looks, a trailing one is dropped by Windows, and
// a space is no part of the name a link is written by.
func trimEnds(name string) string {
	return strings.TrimFunc(name, func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
}

// cutRunes shortens to at most n bytes without splitting a character in half.
func cutRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return s[:n]
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
