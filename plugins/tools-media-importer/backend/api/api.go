// Package api declares the JSON messages of the importer service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "md.numen.tools.media.importer"

// Method names of the service.
const (
	MethodGetLink     = "GetLink"
	MethodReadArticle = "ReadArticle"
	MethodFetchText   = "FetchText"
	MethodCopyMedia   = "CopyMedia"
	MethodDeleteCopy  = "DeleteCopy"
	MethodCreateLinks = "CreateLinks"
)

// GetLinkRequest requests the facts of the link note at Path.
type GetLinkRequest struct {
	Path string `json:"path"`
}

// GetLinkResponse holds the facts of one link.
type GetLinkResponse struct {
	// URL is the web address the note holds.
	URL string `json:"url"`
	// IsCopyable is whether a video is published at the address.
	IsCopyable bool `json:"isCopyable"`
	// IsTextDeleted is whether the text was fetched once and its version deleted since.
	IsTextDeleted bool `json:"isTextDeleted"`
	// Media is the address of the downloaded media the note holds, empty where it holds none.
	Media string `json:"media"`
	// Page is the address of the page that frames the video player the note holds, empty where it
	// holds none.
	Page string `json:"page"`
}

// ReadArticleRequest requests the article text of the link note at Path.
type ReadArticleRequest struct {
	Path string `json:"path"`
}

// ReadArticleResponse holds the article text.
type ReadArticleResponse struct {
	Text string `json:"text"`
}

// FetchTextRequest starts fetching the text of the link note at Path.
type FetchTextRequest struct {
	Path string `json:"path"`
}

// FetchTextResponse is the acknowledgement of a started fetch.
type FetchTextResponse struct{}

// CopyMediaRequest starts copying the media of the link note at Path: the video, or with Kind
// audio the sound alone.
type CopyMediaRequest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// CopyMediaResponse is the acknowledgement of a started copy.
type CopyMediaResponse struct{}

// DeleteCopyRequest deletes the stored video copy of the link note at Path.
type DeleteCopyRequest struct {
	Path string `json:"path"`
}

// DeleteCopyResponse is the acknowledgement of a deleted copy.
type DeleteCopyResponse struct{}

// CreateLinksRequest creates a link note in Folder for each web address in URLs.
type CreateLinksRequest struct {
	Folder string   `json:"folder"`
	URLs   []string `json:"urls"`
}

// CreateLinksResponse lists the notes created, in the order of the addresses; a text that is no web
// address is skipped.
type CreateLinksResponse struct {
	Paths []string `json:"paths"`
}
