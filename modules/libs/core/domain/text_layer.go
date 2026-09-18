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
