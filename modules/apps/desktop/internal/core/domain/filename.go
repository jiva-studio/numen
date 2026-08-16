package domain

import (
	"strings"
	"unicode"
)

// maxFilename is how long a name is allowed to get. Most filesystems stop at
// 255 bytes for one component, and a title is not the place to find that out.
const maxFilename = 120

// Filename is what a note called this is filed under, and whether the title
// survived the trip.
//
// A note is shown by its `title`, else by its first level-one heading, else by
// its filename. So a file named after the title needs no `title` key
// at all — which is what keeps that key read-and-never-written. When the title
// cannot be a filename, `exact` is false and the caller writes the title as a
// heading instead, where the same rule finds it.
func Filename(title string) (name string, exact bool) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(title) {
		switch {
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|':
			// Reserved somewhere that matters, and a slash would make the title
			// a folder.
			b.WriteRune('-')
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
		}
	}

	// A leading dot would file the note where nothing looks, and a
	// trailing dot or space is dropped silently by Windows.
	name = strings.Trim(b.String(), " .")
	if len(name) > maxFilename {
		name = strings.TrimSpace(cutRunes(name, maxFilename))
	}
	if name == "" {
		return "", false
	}
	return name, name == strings.TrimSpace(title)
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
