// Package substitute replaces the variables of a template text.
package substitute

import (
	"strings"
	"time"
)

const (
	defaultDateFormat = "YYYY-MM-DD"
	defaultTimeFormat = "HH:mm"
)

// Variables are the values a template takes. Title is a name; Now is the instant of the call.
type Variables struct {
	Title string
	Now   time.Time
}

// Result is the text after substitution and the place of the first cursor marker.
// Cursor is an offset into Text in UTF-16 code units, -1 for none.
type Result struct {
	Text   string
	Cursor int
}

// variable is a variable read from the start of a text: the text it becomes and the bytes it took.
type variable struct {
	text     string
	length   int
	isCursor bool
}

// Apply replaces the variables of a text. It never fails.
func Apply(text string, vars Variables) Result {
	var out strings.Builder
	cursor := -1
	for rest := text; rest != ""; {
		if found, isVariable := readVariable(rest, vars); isVariable {
			switch {
			case !found.isCursor:
				out.WriteString(found.text)
			case cursor < 0:
				cursor = getUTF16Length(out.String())
			}
			rest = rest[found.length:]
			continue
		}
		out.WriteByte(rest[0])
		rest = rest[1:]
	}
	return Result{Text: out.String(), Cursor: cursor}
}

func readVariable(text string, vars Variables) (variable, bool) {
	if !strings.HasPrefix(text, "{{") {
		return variable{text: "", length: 0, isCursor: false}, false
	}
	inner, _, isClosed := strings.Cut(text[2:], "}}")
	if !isClosed || strings.Contains(inner, "\n") {
		return variable{text: "", length: 0, isCursor: false}, false
	}
	length := len("{{") + len(inner) + len("}}")
	name, format, hasFormat := strings.Cut(inner, ":")
	switch {
	case name == "title" && !hasFormat:
		return variable{text: vars.Title, length: length, isCursor: false}, true
	case name == "cursor" && !hasFormat:
		return variable{text: "", length: length, isCursor: true}, true
	case name == "date":
		return variable{text: formatTime(vars.Now, format, defaultDateFormat), length: length, isCursor: false}, true
	case name == "time":
		return variable{text: formatTime(vars.Now, format, defaultTimeFormat), length: length, isCursor: false}, true
	}
	return variable{text: "", length: 0, isCursor: false}, false
}

func getUTF16Length(text string) int {
	count := 0
	for _, r := range text {
		count++
		if r >= 0x10000 {
			count++
		}
	}
	return count
}
