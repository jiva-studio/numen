// Package api declares the JSON messages of the tags service the plugin's backend serves to its own frontend.
package api

const (
	MethodGetTags = "GetTags"
	MethodSetTags = "SetTags"
)

// GetTagsRequest asks for the tags of the vault.
type GetTagsRequest struct{}

// TagCount is a tag of the vault with the number of notes that have it.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// TagList is every tag of the vault by count, and in Recent the tags most recently set, the latest first, that some
// note still has.
type TagList struct {
	Tags   []TagCount `json:"tags"`
	Recent []string   `json:"recent"`
}

// SetTagsRequest toggles each tag of Toggles on the notes at Paths.
type SetTagsRequest struct {
	Paths   []string `json:"paths"`
	Toggles []string `json:"toggles"`
}

// NoteTags is the tags a note has after SetTags, or why it was not written.
type NoteTags struct {
	Path  string   `json:"path"`
	Tags  []string `json:"tags"`
	Error string   `json:"error,omitempty"`
}

// SetTagsResult holds one NoteTags for each path of the request, in its order.
type SetTagsResult struct {
	Notes []NoteTags `json:"notes"`
}
