// Package api declares the JSON messages of the tags service the plugin's backend serves to its own
// frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.taxonomy"

// MethodGetTagMap is the method that answers the tags of every note that has any.
const MethodGetTagMap = "GetTagMap"

// GetTagMapRequest asks for the tags of every note that has any.
type GetTagMapRequest struct{}

// TagMap is the tags of the notes of the vault by path. A note without tags has no entry.
type TagMap struct {
	Tags map[string][]string `json:"tags"`
}
