package claudecode

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// glimpsed is the value of a named field in JSON that has not finished
// arriving.
//
// Nothing here decodes: half a document does not parse, and waiting for the
// whole of it is waiting for the thing being watched. The first field of that
// name is taken, since the first element of a collection is the one being
// written when there is nothing else to show yet.
func getGlimpse(arguments, field string) string {
	if field == "" {
		return ""
	}
	at := strings.Index(arguments, `"`+field+`"`)
	if at < 0 {
		return ""
	}
	rest := arguments[at+len(field)+2:]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]

	// One way out, so that every value read is read the same way. An escape whose
	// second half has not arrived, and a character cut in half, both end the
	// value where they begin.
	var out strings.Builder
	for i := 0; i < len(rest); i++ {
		if rest[i] == '"' {
			break
		}
		if rest[i] == '\\' {
			if i+1 >= len(rest) {
				break
			}
			out.WriteByte(rest[i])
			i++
		}
		out.WriteByte(rest[i])
	}
	return unquote(whole(out.String()))
}

// whole is the text without a character that has half arrived.
//
// Pieces are cut where the stream cut them, which for anything outside ASCII is
// as likely to be the middle of a character as the end of one.
func whole(text string) string {
	for len(text) > 0 {
		last, size := utf8.DecodeLastRuneInString(text)
		if last != utf8.RuneError || size > 1 {
			return text
		}
		text = text[:len(text)-1]
	}
	return text
}

// unquote turns the escapes of a JSON string into what they stand for.
//
// The last escape may have arrived in pieces — a character named by number is
// six of them, and five are not a character — so the tail is given up a piece at
// a time until what is left reads. An escape shown as itself is text the person
// did not write.
func unquote(text string) string {
	for at := len(text); at > 0; at-- {
		var out string
		if err := json.Unmarshal([]byte(`"`+text[:at]+`"`), &out); err == nil {
			return out
		}
	}
	return ""
}
