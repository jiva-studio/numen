// Package api declares the JSON messages of the importer service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.media.importer"

// Method names of the service.
const (
	MethodGetLink     = "GetLink"
	MethodCopyMedia   = "CopyMedia"
	MethodCreateLinks = "CreateLinks"
)

// GetLinkRequest requests the facts of the link note at Path.
type GetLinkRequest struct {
	Path string `json:"path"`
}

// GetLinkResponse holds the facts of one link.
type GetLinkResponse struct {
	URL        string `json:"url"`
	IsCopyable bool   `json:"isCopyable"`
}

// CopyMediaRequest starts copying the media of the link note at Path: the video, or with Kind
// audio the sound alone.
type CopyMediaRequest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// CopyMediaResponse is the acknowledgment of a started copy.
type CopyMediaResponse struct{}

// CreateLinksRequest creates a link note in Folder for each web address in URLs.
type CreateLinksRequest struct {
	Folder string   `json:"folder"`
	URLs   []string `json:"urls"`
}

// CreateLinksResponse lists the notes of the addresses, in their order: the ones created and the
// ones that existed. A text that is no web address is skipped; Failed holds the addresses whose
// note could be neither found nor created.
type CreateLinksResponse struct {
	Paths  []string `json:"paths"`
	Failed []string `json:"failed"`
}
