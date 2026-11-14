package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrInvalidName is a title no note can be given. Nothing is written.
var ErrInvalidName = errors.New("a note cannot be given this title")

// Filename rejects a title no note can be given, and for every other returns the name
// it is filed under and whether it survived the conversion unchanged.
//
// one carrying a line break. The title comes in trimmed.
func Filename(title string) (name string, exact bool, err error) {
	switch name, exact = GetReducedFilename(title); {
	case name == "":
		return "", false, fmt.Errorf("%w: %q generates empty filename", ErrInvalidName, title)
	case strings.ContainsAny(title, "\n\r"):
		return "", false, fmt.Errorf("%w: %q is more than one line", ErrInvalidName, title)
	}
	return name, exact, nil
}

// maxFilename is how long a name is allowed to get. Most filesystems stop at
// 255 bytes for one component, and a title is not the place to find that out.
const maxFilename = 120

// GetReducedFilename returns the filename a title reduces to, or empty when the
// title reduces to no name.
// When the title cannot be a filename, exact is false and the caller writes the
// title into the `title` key.
// Every returned name is one IsNameable accepts.
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

func trimEnds(name string) string {
	return strings.TrimFunc(name, func(r rune) bool {
		return r == '.' || unicode.IsSpace(r)
	})
}

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

// CheckPath rejects a vault path that a file cannot be given on every supported
// platform: a part holding one of `\ : * ? " < > |`, or ending in a space or a dot.
// A tab key such as `note:dd` is no path, and nothing is written under it.
func CheckPath(path string) error {
	for part := range strings.SplitSeq(path, "/") {
		switch {
		case strings.ContainsAny(part, `\:*?"<>|`):
			return fmt.Errorf("%w: %q holds a character no file name may hold", ErrInvalidName, path)
		case strings.HasSuffix(part, " ") || strings.HasSuffix(part, "."):
			if part == "." || part == ".." {
				continue
			}
			return fmt.Errorf("%w: %q ends in a space or a dot", ErrInvalidName, path)
		}
	}
	return nil
}
