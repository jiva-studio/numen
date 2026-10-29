// Package api holds the JSON messages of the numen.epub service: the requests and responses of the
// methods the plugin's backend serves to its frontend. Field names are the lowerCamelCase keys of the
// JSON; 64-bit integers are JSON numbers (except FileStat.Mtime) and byte strings are base64 text.
package api

// Service is the service name of the plugin's JSON methods, as the manifest declares it.
const Service = "numen.epub"

// FileStat names the bytes of a file: its size and its modification time in nanoseconds. A time of
// zero names no time. The time is a JSON string because a nanosecond count exceeds the integers a
// JavaScript number holds exactly.
type FileStat struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime,string"`
}

// GetBookRequest names the book to open.
type GetBookRequest struct {
	Path string `json:"path"`
}

// GetBookResponse is the metadata and the spine of a book.
type GetBookResponse struct {
	Title        string          `json:"title"`
	Documents    []SpineDocument `json:"documents"`
	Parts        []BookPart      `json:"parts"`
	PrintedPages []PrintedPage   `json:"printedPages"`
	PageCount    int             `json:"pageCount"`
	TextBytes    int             `json:"textBytes"`
	PageBytes    int             `json:"pageBytes"`
	// Stat names the file the book was read from, and is absent where the backend names none.
	Stat *FileStat `json:"stat,omitempty"`
}

// ReadBookMarkupRequest names a document of a book's spine.
type ReadBookMarkupRequest struct {
	Path     string `json:"path"`
	Document string `json:"document"`
	// Baseline names the bytes the caller read the book from, and is absent for any bytes.
	Baseline *FileStat `json:"baseline,omitempty"`
}

// ReadBookMarkupResponse is the rendered HTML of one document.
type ReadBookMarkupResponse struct {
	Markup string `json:"markup"`
}

// ReadBookEntryRequest names one entry of a book's archive.
type ReadBookEntryRequest struct {
	Path  string `json:"path"`
	Entry string `json:"entry"`
	// Baseline names the bytes the entry must be read from, and is absent for any bytes.
	Baseline *FileStat `json:"baseline,omitempty"`
}

// ReadBookEntryResponse is the bytes of one entry, as base64 text in JSON.
type ReadBookEntryResponse struct {
	Body []byte `json:"body"`
}

// GetCfiRequest names a character of a book's text.
type GetCfiRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
}

// GetCfiResponse is the EPUB CFI of the character.
type GetCfiResponse struct {
	Cfi string `json:"cfi"`
}

// FindOffsetRequest names an EPUB CFI of a book.
type FindOffsetRequest struct {
	Path string `json:"path"`
	Cfi  string `json:"cfi"`
}

// FindOffsetResponse is the offset of the book's text the CFI names.
type FindOffsetResponse struct {
	Offset int `json:"offset"`
}

// SpineDocument is one document of the spine and the range of the text it covers.
type SpineDocument struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

// BookPart is one section or chapter a book names.
type BookPart struct {
	Title  string `json:"title"`
	Offset int    `json:"offset"`
	Level  int    `json:"level"`
}

// PrintedPage is one page label of a printed edition.
type PrintedPage struct {
	Label  string `json:"label"`
	Offset int    `json:"offset"`
}
