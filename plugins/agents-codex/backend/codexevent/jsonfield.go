package codexevent

import "encoding/json"

func getString(object map[string]any, key string) string {
	value, ok := object[key].(string)
	if !ok {
		return ""
	}
	return value
}

func getMap(object map[string]any, key string) map[string]any {
	value, ok := object[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

// DecodeLine parses one output line as a JSON object and reports whether the line was one.
func DecodeLine(line []byte) (map[string]any, bool) {
	var event map[string]any
	if err := json.Unmarshal(line, &event); err != nil {
		return nil, false
	}
	return event, true
}
