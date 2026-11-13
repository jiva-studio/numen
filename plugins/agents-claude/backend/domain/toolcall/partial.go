package toolcall

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// ReadPartialText returns the text of the field of JSON arguments that are still arriving, as far as it
// has arrived: the value stops at an escape or a character cut off at the end. It is empty where the field
// is absent or holds no text.
func ReadPartialText(arguments, field string) string {
	start := strings.Index(arguments, `"`+field+`"`)
	if start < 0 {
		return ""
	}
	rest := strings.TrimLeft(arguments[start+len(field)+2:], " \t\r\n")
	rest, isField := strings.CutPrefix(rest, ":")
	if !isField {
		return ""
	}
	rest, isText := strings.CutPrefix(strings.TrimLeft(rest, " \t\r\n"), `"`)
	if !isText {
		return ""
	}
	end := 0
	for end < len(rest) && rest[end] != '"' {
		if rest[end] == '\\' {
			end++
		}
		end++
	}
	return decodePartialText(rest[:min(end, len(rest))])
}

func decodePartialText(quoted string) string {
	for end := len(quoted); end > 0; end-- {
		if !utf8.ValidString(quoted[:end]) {
			continue
		}
		var text string
		if json.Unmarshal([]byte(`"`+quoted[:end]+`"`), &text) == nil {
			return text
		}
	}
	return ""
}
