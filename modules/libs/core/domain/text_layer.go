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
