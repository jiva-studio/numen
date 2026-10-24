package markup

// A Markup is one spine document as the elements a book is drawn from, with
// document and reading its text are one walk, so an offset here is the offset a
// chunk of that text carries.
type Markup struct {
	// Path is the document's name inside the archive.
	Path string
	// Offset is where the document's text begins in the book's Text, and Length
	// how many bytes of it the document is.
	Offset int
	Length int
	// Nodes are what the document holds, in reading order.
	Nodes []Node
}

// A Node is one element of a document, or one run of its text.
type Node struct {
	// Name is the element's name, and empty for a run of text.
	Name string
	// Text is the run's text, and empty for an element.
	Text string
	// Offset is where the node begins in the book's Text.
	Offset     int
	Attributes []Attribute
	Children   []Node
}

// An Attribute is one attribute of an element, of the few an element may carry.
type Attribute struct {
	Name  string
	Value string
}

// Shift moves a document to at its current position in the book.
func Shift(nodes []Node, delta int) {
	for i := range nodes {
		nodes[i].Offset += delta
		Shift(nodes[i].Children, delta)
	}
}
