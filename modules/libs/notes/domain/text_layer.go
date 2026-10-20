package domain

// A PartStart is somewhere in a source's text that carries a name. Parts bound
// the divisions a text is cut inside, and need not arrive in order.
type PartStart struct {
	Title  string
	Offset int
}

// A TextLayer is the text a document carries of its own, taken out. It is
// deterministic and stored nowhere, and what a model reads off the same pages
// is a reading and an artifact instead.
type TextLayer struct {
	// Text is every page's text in the order the document is paginated, as one
	// stream. Every offset below is an offset into it.
	Text string
	// Parts are the names the document gives divisions of itself.
	Parts []PartStart
	// Pages is where each page begins.
	Pages []int
}

// A BookPage is a page of the printed edition a book was set from, at the
// offset of its text where it starts.
type BookPage struct {
	Label  string
	Offset int
}

// A BookText is the text a book that reflows carries, taken out. Every offset is
// a byte offset into Text, and Parts and Pages ascend by offset.
type BookText struct {
	Text string
	// Parts are the names the book gives divisions of itself.
	Parts []PartStart
	// Pages are the pages of the printed edition the book names.
	Pages []BookPage
}

// A BookSection is one document of a book's spine, and where its text stands in
// the book's text.
type BookSection struct {
	// Path is the section's name inside the archive.
	Path string
	// Offset is where the section's text begins, and Length how many bytes of
	// the book's text it is.
	Offset int
	Length int
}

// A BookPart is a named division of a book, at the offset its text begins.
type BookPart struct {
	Title  string
	Offset int
	// Level is the depth of the heading the part came from, and zero for a part
	// the book's own navigation named.
	Level int
}

// A BookShape is what a book that reflows is: what it is called, the sections it
// is read in, what it names inside them and how far its text runs. Every offset
// and length is counted in bytes of the book's text.
type BookShape struct {
	Title string
	// Sections are the documents of the spine, in reading order.
	Sections []BookSection
	// Parts are the names the book gives divisions of itself, ascending by offset.
	Parts []BookPart
	// Pages are the pages of the printed edition the book was set from, ascending
	// by offset.
	Pages []BookPage
	// PageCount is how many pages the book is read in, counted over its text.
	PageCount int
	// TextBytes is how many bytes the book's text is, and PageBytes how many of
	// them stand on one page.
	TextBytes int
	PageBytes int
	// Fingerprint is the file the book was read from.
	Fingerprint Fingerprint
}
