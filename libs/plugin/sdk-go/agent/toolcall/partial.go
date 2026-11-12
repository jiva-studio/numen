package toolcall

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// ReadPartialField reads the value of a named field in JSON that has not finished arriving.
func ReadPartialField(arguments, field string) string {
	if field == "" {
		return ""
	}
	idx := strings.Index(arguments, `"`+field+`"`)
	if idx < 0 {
		return ""
	}
	rest := arguments[idx+len(field)+2:]
	rest = strings.TrimLeft(rest, " \t\r\n")
	if !strings.HasPrefix(rest, ":") {
		return ""
	}
	rest = strings.TrimLeft(rest[1:], " \t\r\n")
	if !strings.HasPrefix(rest, `"`) {
		return ""
	}
	rest = rest[1:]

	var out strings.Builder
	for i := 0; i < len(rest); i++ {
		// gomutants:disable-next-line BRANCH_IF reason="an unescaped quote in out fails json.Unmarshal in unquote until pos backs up to this index"
		if rest[i] == '"' {
			break
		}
		if rest[i] == '\\' {
			if i+1 >= len(rest) {
				// gomutants:disable-next-line INVERT_LOOP_CTRL reason="i+1 >= len(rest) holds only on the final byte, where continue and break both end the loop"
				break
			}
			out.WriteByte(rest[i])
			i++
		}
		out.WriteByte(rest[i])
	}
	return unquote(trimPartialRune(out.String()))
}

func trimPartialRune(text string) string {
	for len(text) > 0 {
		last, size := utf8.DecodeLastRuneInString(text)
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_INCREMENT,INTEGER_DECREMENT reason="utf8.DecodeLastRuneInString returns size 1 for invalid UTF-8 bytes and size 3 for a valid U+FFFD rune"
		if last != utf8.RuneError || size > 1 {
			return text
		}
		text = text[:len(text)-1]
	}
	return ""
}

func unquote(text string) string {
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_INCREMENT,INTEGER_DECREMENT reason="at pos 0 text[:0] unmarshals to the same empty string that the loop fallback returns"
	for pos := len(text); pos > 0; pos-- {
		var out string
		if err := json.Unmarshal([]byte(`"`+text[:pos]+`"`), &out); err == nil {
			return out
		}
	}
	return ""
}
