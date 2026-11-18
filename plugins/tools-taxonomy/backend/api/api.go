// Package api declares the JSON messages of the tags service the plugin's backend serves to its own
// frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.taxonomy"

const (
	MethodGetTagMap   = "GetTagMap"
	MethodWatchTagMap = "WatchTagMap"
)

// GetTagMapRequest asks for the tags of every note that has any.
type GetTagMapRequest struct{}

// TagMap is the tags of the notes of the vault by path. A note without tags has no entry.
type TagMap struct {
	Tags map[string][]string `json:"tags"`
}

// WatchTagMapRequest asks to follow the tags of the notes of the vault.
type WatchTagMapRequest struct{}

// TagMapMessage is one message of the stream WatchTagMap: the first holds the whole map in Tags, each
// later one holds in Changed the paths whose tags changed with their tags now; a path left with no
// tags, removed or moved away has an empty list.
type TagMapMessage struct {
	Tags    map[string][]string `json:"tags,omitempty"`
	Changed map[string][]string `json:"changed,omitempty"`
}
