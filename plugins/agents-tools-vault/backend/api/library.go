package api

type LibrarySearchInput struct {
	Query  string   `json:"query" jsonschema:"words to search for, as they appear in the text"`
	Limit  int      `json:"limit,omitempty" jsonschema:"how many passages to return (default: 20, max: 100)"`
	Kinds  []string `json:"kinds,omitempty" jsonschema:"the kinds of file to search: note for markdown notes, file for every other file (books, PDFs, recordings, scans); all kinds when left out"`
	Folder string   `json:"folder,omitempty" jsonschema:"limit the result to this folder and its subfolders, relative to the vault folder; the whole vault when left out"`
}

// Place is a place in a source: a page of a PDF, a position or range in a recording, or a position in
// an EPUB.
type Place struct {
	Page            int      `json:"page,omitempty" jsonschema:"PDF: the page, from 1"`
	PageCount       int      `json:"pageCount,omitempty" jsonschema:"PDF: the number of pages"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty" jsonschema:"recording: seconds from the start"`
	EndSeconds      *float64 `json:"endSeconds,omitempty" jsonschema:"recording: seconds from the start to the end of the range"`
	CFI             string   `json:"cfi,omitempty" jsonschema:"EPUB: the CFI of the place"`
	Chapter         string   `json:"chapter,omitempty" jsonschema:"EPUB: the table-of-contents title at the place"`
}

// LibraryMatch is a passage that matches a library search.
type LibraryMatch struct {
	Path   string `json:"path" jsonschema:"the path of the source file, relative to the vault folder"`
	Title  string `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Place  *Place `json:"place,omitempty" jsonschema:"the place of the passage in the source; absent for a source with no pages, times or EPUB positions, such as a note"`
	Text   string `json:"text" jsonschema:"the text around the hit, truncated to a fixed size, often mid-sentence"`
	Start  int    `json:"start" jsonschema:"the byte offset of the passage in the text of the file"`
	Length int    `json:"length" jsonschema:"the length of the passage in bytes"`
	Line   int    `json:"line" jsonschema:"the line of the hit, counted from 1 at the first line of the text; for a note, the first line of its body"`
	URI    string `json:"uri" jsonschema:"the address of this passage; use it as the target of a markdown link to the passage"`
}

type LibrarySearchOutput struct {
	Query   string         `json:"query" jsonschema:"the query searched for"`
	Matches []LibraryMatch `json:"matches" jsonschema:"the matching passages, best first"`
}

type LibraryReadInput struct {
	Path   string `json:"path" jsonschema:"the file's path relative to the vault folder"`
	Page   int    `json:"page,omitempty" jsonschema:"the page to read, counted from 1; for a source whose text has pages"`
	Start  int    `json:"start,omitempty" jsonschema:"the byte offset in the text to start reading from; pass the nextStart of the previous result to continue"`
	Length int    `json:"length,omitempty" jsonschema:"bytes of text to read: default 20000 and at most 50000 for a text version; at most 16000 for a plain text file"`
}

// LibraryRange is a range of the text and the place in the source where it starts.
type LibraryRange struct {
	Start  int    `json:"start" jsonschema:"the byte offset of the range in the text"`
	Length int    `json:"length" jsonschema:"the length of the range in bytes"`
	Place  *Place `json:"place" jsonschema:"the place in the source where the range starts"`
}

type LibraryReadOutput struct {
	Path      string         `json:"path" jsonschema:"the path of the file, relative to the vault folder"`
	Kind      string         `json:"kind,omitempty" jsonschema:"the kind of the text version, as set by the plugin that created it; absent for a text file"`
	Version   string         `json:"version,omitempty" jsonschema:"the text version read"`
	Text      string         `json:"text" jsonschema:"the text read"`
	Start     int            `json:"start" jsonschema:"the byte offset of the returned text in the whole text"`
	Length    int            `json:"length" jsonschema:"the length of the returned text in bytes"`
	Size      int            `json:"size" jsonschema:"the length of the whole text in bytes"`
	NextStart *int           `json:"nextStart,omitempty" jsonschema:"the start to read next; absent at the end of the text"`
	Ranges    []LibraryRange `json:"ranges,omitempty" jsonschema:"the ranges of the returned text with their places in the source, in order"`
}
