// Package pending is the content of the file of the plugin's store folder that keeps the files whose
// run has not ended, by the number the index keeps each file under.
package pending

import "encoding/json"

const version = 1

type file struct {
	Version int              `json:"version"`
	Files   map[int64]string `json:"files"`
}

// Parse returns the files the content keeps, and false for content that is not of this version.
func Parse(content []byte) (map[int64]string, bool) {
	var stored file
	if json.Unmarshal(content, &stored) != nil || stored.Version != version {
		return nil, false
	}
	if stored.Files == nil {
		stored.Files = map[int64]string{}
	}
	return stored.Files, true
}

// Encode returns the content that keeps the files.
func Encode(files map[int64]string) ([]byte, error) {
	return json.Marshal(file{Version: version, Files: files})
}
