// Package tagfile is the content of tags.json: the tags of each note that has any, by the number the
// index keeps the note under.
package tagfile

import "encoding/json"

const version = 2

// Note is the tags of one note and where it is now.
type Note struct {
	Path string   `json:"path"`
	Tags []string `json:"tags"`
}

type file struct {
	Version int            `json:"version"`
	Notes   map[int64]Note `json:"notes"`
}

// Parse returns the notes the content keeps, and nil for content that is not of this version.
func Parse(content []byte) map[int64]Note {
	var stored file
	if json.Unmarshal(content, &stored) != nil || stored.Version != version {
		return nil
	}
	return stored.Notes
}

// Encode returns the content that keeps the notes.
func Encode(notes map[int64]Note) ([]byte, error) {
	return json.Marshal(file{Version: version, Notes: notes})
}
