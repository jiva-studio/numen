// Package service serves the PDF reader plugin's text layer and JSON methods.
package service

// APIService is the service name the plugin's JSON methods are mounted under. The manifest
// `services` declares the same name.
const APIService = "numen.pdf"

// FileStat identifies the bytes of a file.
type FileStat struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"`
}

// Range is a run of a text version, as byte offsets from Start up to End.
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Page is the size of one page.
type Page struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// GetDocumentRequest names the PDF file to describe.
type GetDocumentRequest struct {
	Path string `json:"path"`
}

// GetDocumentResponse lists the pages of a PDF file in order.
type GetDocumentResponse struct {
	Pages []Page    `json:"pages"`
	Stat  *FileStat `json:"stat,omitempty"`
}

// GetHighlightsRequest names a file and the runs of its live text version to place.
type GetHighlightsRequest struct {
	Path   string  `json:"path"`
	Ranges []Range `json:"ranges"`
}

// GetHighlightsResponse holds one run for each requested range, in request order.
type GetHighlightsResponse struct {
	Runs []DocumentRun `json:"runs"`
}

// DocumentRun is where one range of text is placed, one place for each page.
type DocumentRun struct {
	Places []DocumentPlace `json:"places"`
}

// DocumentPlace is the rectangles of a run on one page.
type DocumentPlace struct {
	Page  int            `json:"page"`
	Rects []DocumentRect `json:"rects"`
}

// DocumentRect is a rectangle on a page, in fractions of the page size.
type DocumentRect struct {
	MinX float32 `json:"minX"`
	MinY float32 `json:"minY"`
	MaxX float32 `json:"maxX"`
	MaxY float32 `json:"maxY"`
}
