// Package toolcall reads what a tool call is about out of its arguments, for a person who watches the
// call.
package toolcall

import "encoding/json"

// ReadSubject returns what the call with the JSON arguments is about, and the path it works on: the
// path is the first text of "path" or "file", and the subject the first text of "query", "about" or
// "title", else the path. Both are empty where the arguments say neither.
func ReadSubject(arguments []byte) (about, path string) {
	var parsed map[string]any
	if json.Unmarshal(arguments, &parsed) != nil {
		return "", ""
	}
	path = readFirstText(parsed, "path", "file")
	about = readFirstText(parsed, "query", "about", "title")
	if about == "" {
		about = path
	}
	return about, path
}

func readFirstText(parsed map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, isText := parsed[key].(string); isText && value != "" {
			return value
		}
	}
	return ""
}
