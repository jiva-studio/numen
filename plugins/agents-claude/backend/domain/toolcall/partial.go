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
	_, rest, _ := strings.Cut(arguments, `"`+field+`"`)
	rest, isField := strings.CutPrefix(strings.TrimLeft(rest, " \t\r\n"), ":")
	if !isField {
		return ""
	}
	rest, isText := strings.CutPrefix(strings.TrimLeft(rest, " \t\r\n"), `"`)
	if !isText {
		return ""
	}
	return decodePartialText(rest)
}

func decodePartialText(quoted string) string {
	for quoted != "" {
		if utf8.ValidString(quoted) {
			var text string
			if json.Unmarshal([]byte(`"`+quoted+`"`), &text) == nil {
				return text
			}
		}
		quoted = quoted[:len(quoted)-1]
	}
	return ""
}
