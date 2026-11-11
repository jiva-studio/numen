package markup

// A Markup is one spine document as the elements a book is drawn from, with
// document and reading its text are one walk, so an offset here is the offset a
// chunk of that text carries.
type Markup struct {
	Path   string
	Offset int
	Length int
	Nodes  []Node
}

// A Node is one element of a document, or one run of its text.
type Node struct {
	Name       string
	Text       string
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
