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
	return unquote(trimPartialRune(out.String()))
}

func trimPartialRune(text string) string {
	for len(text) > 0 {
		last, size := utf8.DecodeLastRuneInString(text)
		if last != utf8.RuneError || size > 1 {
			return text
		}
		text = text[:len(text)-1]
	}
	return text
}

func unquote(text string) string {
	for at := len(text); at > 0; at-- {
		var out string
		if err := json.Unmarshal([]byte(`"`+text[:at]+`"`), &out); err == nil {
			return out
		}
	}
	return ""
}
