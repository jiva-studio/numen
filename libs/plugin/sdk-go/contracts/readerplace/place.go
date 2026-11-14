// Package readerplace is the contract of the place a reader tab shows. A reader reports it to the window
// as JSON in this shape; the core passes it on unread, and a plugin reads it from the Place of a
// host.OpenTab.
package readerplace

// Place is the place a reader tab shows. A field the reader does not have is left out of the JSON.
//
//numen:api
type Place struct {
	// Page is the page a PDF shows, from 1.
	Page int `json:"page,omitempty"`
	// PageCount is the number of pages of a PDF.
	PageCount int `json:"pageCount,omitempty"`
	// PositionSeconds is the time of a recording, in seconds from its start; nil for a tab that is not a
	// recording.
	PositionSeconds *float64 `json:"positionSeconds,omitempty"`
	// CFI is the EPUB CFI of the place shown.
	CFI string `json:"cfi,omitempty"`
	// Chapter is the table-of-contents title of an EPUB at the place shown.
	Chapter string `json:"chapter,omitempty"`
}
