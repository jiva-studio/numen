package service

// getString returns the string at key, or "" where the key is absent or holds another type.
func getString(object map[string]any, key string) string {
	value, ok := object[key].(string)
	if !ok {
		return ""
	}
	return value
}

// getMap returns the object at key, or nil where the key is absent or holds another type.
func getMap(object map[string]any, key string) map[string]any {
	value, ok := object[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}
