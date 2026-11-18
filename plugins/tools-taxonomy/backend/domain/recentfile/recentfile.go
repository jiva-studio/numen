// Package recentfile is the content of recent.json: the tags most recently set, the latest first.
package recentfile

import "encoding/json"

// Parse returns the tags the content keeps, and nil for content that is not a list of tags.
func Parse(content []byte) []string {
	var list []string
	if json.Unmarshal(content, &list) != nil {
		return nil
	}
	return list
}

// Encode returns the content that keeps the tags.
func Encode(list []string) ([]byte, error) {
	return json.Marshal(list)
}
