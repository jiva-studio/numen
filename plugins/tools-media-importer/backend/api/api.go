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
	MethodCopyVideo   = "CopyVideo"
	MethodDeleteCopy  = "DeleteCopy"
	MethodCreateLinks = "CreateLinks"
)

// GetLinkRequest requests the facts of the link note at Path.
type GetLinkRequest struct {
	Path string `json:"path"`
}

// GetLinkResponse holds the facts of one link.
type GetLinkResponse struct {
	URL           string `json:"url"`
	IsCopyable    bool   `json:"isCopyable"`
	CopySize      int64  `json:"copySize"`
	IsTextDeleted bool   `json:"isTextDeleted"`
	EmbedPage     string `json:"embedPage"`
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
	Path          string `json:"path"`
	IsUserStarted bool   `json:"isUserStarted"`
}

// FetchTextResponse is the acknowledgement of a started fetch.
type FetchTextResponse struct{}

// CopyVideoRequest starts copying the video of the link note at Path.
type CopyVideoRequest struct {
	Path string `json:"path"`
}

// CopyVideoResponse is the acknowledgement of a started copy.
type CopyVideoResponse struct{}

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
