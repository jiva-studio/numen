package domain

import (
	"maps"
	"path"
	"slices"
	"strings"
)

// Recordings are the containers a recording is kept in, each against the media
// type it is played as.
//
// One table answers three questions that have to agree: which files a vault
// walks as recordings, what would read one, and what a player is told it is
// being given. A container added to one list alone is a source the vault holds
// and the player refuses.
var Recordings = map[string]string{
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".flac": "audio/flac",
}

// MediaType is what a file of this name is played as, and nothing for a name no
// recording is kept under.
func MediaType(name string) string {
	return Recordings[strings.ToLower(path.Ext(name))]
}

// RecordingExtensions are the containers, in the order a person reads a list.
func RecordingExtensions() []string {
	return slices.Sorted(maps.Keys(Recordings))
}
