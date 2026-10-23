package name

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
// one carrying a line break. The title comes in trimmed.
func Filename(title string) (name string, exact bool, err error) {
	switch name, exact = GetReducedFilename(title); {
	case name == "":
		return "", false, fmt.Errorf("%w: %q generates empty filename", ErrUnnameable, title)
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
	name = trimEnds(sanitizeRunes(title))
	if len(name) > maxFilename {
		name = trimEnds(truncateAtRuneBoundary(name, maxFilename))
	}
	if name == "" {
		return "", false
	}
	if isDevice(name) {
		return name + "-", false
	}
	return name, name == strings.TrimSpace(title)
}

func sanitizeRunes(title string) string {
	var b strings.Builder
	var last rune
	for _, r := range strings.TrimSpace(title) {
		if unicode.IsControl(r) {
			continue
		}
		if isReservedRune(r) {
			r = '-'
		} else if isDuplicateBracket(r, last) {
			continue
		}
		b.WriteRune(r)
		last = r
	}
	return b.String()
}

func isReservedRune(r rune) bool {
	switch r {
	case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '#':
		return true
	default:
		return false
	}
}

func isDuplicateBracket(r, last rune) bool {
	return (r == '[' || r == ']') && r == last
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

// truncateAtRuneBoundary shortens to at most n bytes without splitting a character in half.
func truncateAtRuneBoundary(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8Start(s[n]) {
		n--
	}
	return s[:n]
}

func utf8Start(b byte) bool { return b&0xC0 != 0x80 }
