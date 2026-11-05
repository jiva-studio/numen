package service

import "github.com/jiva-studio/numen/libs/plugin-sdk-go/host"

// LibrarySearchInput is the input of the library_search tool.
type LibrarySearchInput struct {
	Query      string   `json:"query" jsonschema:"words to look for, in the words of the text itself"`
	Limit      int      `json:"limit,omitempty" jsonschema:"how many passages to return (default: 20, max: 100)"`
	Kinds      []string `json:"kinds,omitempty" jsonschema:"which sorts of file to look in: note for markdown notes, file for every other file (books, PDFs, recordings, scans); all of them when left out"`
	PathPrefix string   `json:"path_prefix,omitempty" jsonschema:"only look in the files whose path starts with this, relative to the vault folder, such as library/"`
}

// LibraryMatch is a passage that answers a library search.
type LibraryMatch struct {
	Source   string `json:"source" jsonschema:"the file the text is read from, relative to the vault folder"`
	Title    string `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Location string `json:"location,omitempty" jsonschema:"where this sits in the source's own numbering, such as a page, a chapter or a time; absent when the format offers none"`
	Text     string `json:"text" jsonschema:"the words around the hit, cut to a size; it ends where it was cut, often mid-sentence"`
	Start    int    `json:"start" jsonschema:"where the passage begins in the source's text, in bytes"`
	Length   int    `json:"length" jsonschema:"how long the passage is, in bytes"`
	Line     int    `json:"line,omitempty" jsonschema:"the line of the hit in a note, counted from the first line of its prose"`
	At       string `json:"at" jsonschema:"the address of this passage; write it into your answer as a markdown link where you speak about the passage, so that the person can go to it"`
}

// LibrarySearchOutput is the output of the library_search tool.
type LibrarySearchOutput struct {
	Query   string         `json:"query"`
	Matches []LibraryMatch `json:"matches"`
}

// LibraryReadInput is the input of the library_read tool.
type LibraryReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault folder"`
	Page   int    `json:"page,omitempty" jsonschema:"read this page, counted from 1; for a source whose text has pages"`
	Start  int    `json:"start,omitempty" jsonschema:"byte offset in the text to start reading from; the next of the previous answer continues it"`
	Length int    `json:"length,omitempty" jsonschema:"how many bytes of text to read; at most 50000, and 20000 when left out"`
}

// LibraryLocation is a stretch of the text and the place the source numbers it by.
type LibraryLocation struct {
	Start       int     `json:"start" jsonschema:"where the stretch begins in the text, in bytes"`
	Length      int     `json:"length" jsonschema:"how long the stretch is, in bytes"`
	Page        int     `json:"page,omitempty" jsonschema:"the page, counted from 1"`
	CFI         string  `json:"cfi,omitempty" jsonschema:"the position in a book"`
	FromSeconds float64 `json:"from_seconds,omitempty" jsonschema:"where in the recording the stretch is spoken"`
	ToSeconds   float64 `json:"to_seconds,omitempty" jsonschema:"where in the recording the stretch ends"`
}

// LibraryReadOutput is the output of the library_read tool.
type LibraryReadOutput struct {
	Path      string            `json:"path"`
	Kind      string            `json:"kind,omitempty" jsonschema:"what the text is, as the plugin that made it names it; absent for a text file"`
	Version   string            `json:"version,omitempty" jsonschema:"the text version read"`
	Text      string            `json:"text"`
	Start     int               `json:"start" jsonschema:"where the text returned begins in the whole text, in bytes"`
	Length    int               `json:"length" jsonschema:"how many bytes were returned"`
	Size      int               `json:"size" jsonschema:"how long the whole text is, in bytes"`
	Next      *int              `json:"next,omitempty" jsonschema:"the start to read next; absent at the end of the text"`
	Locations []LibraryLocation `json:"locations,omitempty" jsonschema:"the places the returned text stands in, in order"`
}

// WithLibrary gives the tools the text versions and the search index of the vaults, which the tools
// that read the library need.
func (t *Tools) WithLibrary(texts host.Texts, search host.Search) *Tools {
	t.texts, t.search = texts, search
	return t
}
