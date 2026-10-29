// Package api declares the JSON messages of the importer service the plugin's backend serves to its
// own frontend. Keys are lowerCamelCase; a failure is a server.APIError, not a message field.
package api

// Service is the service name the manifest declares.
const Service = "numen.importer"

// Method names of the service.
const (
	MethodGetLink     = "GetLink"
	MethodReadArticle = "ReadArticle"
	MethodFetchText   = "FetchText"
	MethodCopyVideo   = "CopyVideo"
	MethodDeleteCopy  = "DeleteCopy"
	MethodCreateLinks = "CreateLinks"
)

// GetLinkRequest asks for the facts of the link note at Path.
type GetLinkRequest struct {
	Path string `json:"path"`
}

// GetLinkResponse holds the facts of one link.
type GetLinkResponse struct {
	// URL is the web address the note holds.
	URL string `json:"url"`
	// IsCopyable is whether a video is published at the address.
	IsCopyable bool `json:"isCopyable"`
	// CopySize is the byte count of the stored copy, zero where none is kept.
	CopySize int64 `json:"copySize"`
	// IsTextDeleted is whether the text was fetched once and its version deleted since.
	IsTextDeleted bool `json:"isTextDeleted"`
	// EmbedPage is the page below the plugin route that frames the video player, empty where none is known.
	EmbedPage string `json:"embedPage"`
}

// ReadArticleRequest asks for the article text of the link note at Path.
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
	// IsUserStarted is whether the person asked for the fetch; a fetch they did not ask for leaves a
	// deleted text deleted.
	IsUserStarted bool `json:"isUserStarted"`
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
