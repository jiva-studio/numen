package hostchat

import "encoding/json"

func readSubject(arguments string) (about, path string) {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
		return "", ""
	}
	path = readFirstString(parsed, "path", "file")
	about = readFirstString(parsed, "query", "about", "title")
	if about == "" {
		about = path
	}
	return about, path
}

func readFirstString(parsed map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, isText := parsed[key].(string); isText && value != "" {
			return value
		}
	}
	return ""
}
