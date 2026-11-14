package codexevent

import "encoding/json"

func getString(object map[string]any, key string) string {
	if value, ok := object[key].(string); ok {
		return value
	}
	return ""
}

func getMap(object map[string]any, key string) map[string]any {
	if value, ok := object[key].(map[string]any); ok {
		return value
	}
	return nil
}

// DecodeLine parses one output line as a JSON object and reports whether the line was one.
func DecodeLine(line []byte) (map[string]any, bool) {
	var event map[string]any
	if err := json.Unmarshal(line, &event); err != nil {
		return nil, false
	}
	return event, true
}
