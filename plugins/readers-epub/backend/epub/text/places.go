package text

import (
	"path"
	"slices"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Places are where the block elements of one document begin in its text, and the text node each
// part of its text was written from. The offsets count from the start of the document, as ReadNodes
// counts them.
type Places struct {
	// Blocks are where each block element begins, ascending.
	Blocks []int
	// Runs are the text nodes that wrote text, in document order.
	Runs []Run
}

// A Run is one text node of a document and the text it wrote.
type Run struct {
	// Start and End are where the node's text begins and ends in the text.
	Start, End int
	// Steps are the EPUB CFI steps from the root element to the node. The last step is odd: the
	// node is character data between elements.
	Steps []int
	// Base is how many UTF-16 code units the text nodes of the same run of character data hold
	// before this node. A comment splits one run into two nodes.
	Base int

	data    string
	isPre   bool
	last    byte
	hasLast bool
}

// ReadPlaces reads one document and returns its places.
func ReadPlaces(docPath string, raw []byte) Places {
	x := NewExtractor()
	x.markup = newBuilder(path.Dir(docPath))
	x.isPlacing = true
	x.ReadDocument(docPath, raw)
	x.places.Blocks = slices.Compact(x.places.Blocks)
	return x.places
}

// recordRun records the text node that wrote the text from start, with the byte before it.
func (x *Extractor) recordRun(n *html.Node, start int, last byte, hasLast bool) {
	steps, base := getTextSteps(n)
	x.places.Runs = append(x.places.Runs, Run{
		Start: start, End: len(x.out), Steps: steps, Base: base,
		data: n.Data, isPre: x.preformattedDepth > 0, last: last, hasLast: hasLast,
	})
}

// GetCharOffset is the character offset in the node's run of character data of the character that
// wrote the text at an offset, or of the first one written after it.
func (r Run) GetCharOffset(offset int) int {
	found := -1
	end := r.walk(func(char, out, n int) bool {
		if n > 0 && out+n > offset {
			found = char
			return false
		}
		return true
	})
	if found < 0 {
		return r.Base + end
	}
	return r.Base + found
}

// FindOffset is where the character at a character offset of the node's run of character data is
// written in the text. A character the text drops, such as a second space, is where the next one
// is written.
func (r Run) FindOffset(char int) int {
	found := -1
	r.walk(func(index, out, _ int) bool {
		if r.Base+index >= char {
			found = out
			return false
		}
		return true
	})
	if found < 0 {
		return r.End
	}
	return found
}

// Length is how many UTF-16 code units the node holds.
func (r Run) Length() int {
	return r.walk(func(int, int, int) bool { return true })
}

// walk replays how the node's characters were written: for each, its offset in UTF-16 code units,
// where it was written, and how many bytes it wrote. It stops when visit returns false, and returns
// the code units walked.
func (r Run) walk(visit func(char, out, n int) bool) int {
	char, out, last, hasLast := 0, r.Start, r.last, r.hasLast
	for i := 0; i < len(r.data); {
		ch, size := utf8.DecodeRuneInString(r.data[i:])
		n := 0
		switch {
		case r.isPre:
			n, last = size, r.data[i+size-1]
		case unicode.IsSpace(ch):
			if hasLast && last != ' ' && last != '\n' {
				n, last = 1, ' '
			}
		default:
			// An invalid byte is written as U+FFFD.
			encoded := utf8.AppendRune(nil, ch)
			n, last = len(encoded), encoded[len(encoded)-1]
		}
		hasLast = hasLast || n > 0
		if !visit(char, out, n) {
			return char
		}
		out += n
		char += utf16Len(ch)
		i += size
	}
	return char
}

func utf16Len(ch rune) int {
	if ch >= 0x10000 {
		return 2
	}
	return 1
}

// getElementSteps is the EPUB CFI steps from the root element of a document to an element: each the
// even index of the element among its parent's children, counting elements only (EPUB CFI 1.1
// §3.1.3).
func getElementSteps(n *html.Node) []int {
	var steps []int
	for node := n; node.Parent != nil && node.Parent.Type == html.ElementNode; node = node.Parent {
		steps = append(steps, 2+2*countElementsBefore(node))
	}
	slices.Reverse(steps)
	return steps
}

// getTextSteps is the EPUB CFI steps to a text node, ending in the odd index of its run of
// character data, and how many UTF-16 code units the nodes of that run hold before it.
func getTextSteps(n *html.Node) ([]int, int) {
	var steps []int
	if n.Parent != nil && n.Parent.Type == html.ElementNode {
		steps = getElementSteps(n.Parent)
	}
	base := 0
	for before := n.PrevSibling; before != nil && before.Type != html.ElementNode; before = before.PrevSibling {
		if before.Type == html.TextNode {
			base += Run{data: before.Data, isPre: true}.Length()
		}
	}
	return append(steps, 1+2*countElementsBefore(n)), base
}

func countElementsBefore(n *html.Node) int {
	count := 0
	for before := n.PrevSibling; before != nil; before = before.PrevSibling {
		if before.Type == html.ElementNode {
			count++
		}
	}
	return count
}

// writeTextNode writes the text of a text node, and records the node where places are read.
func (x *Extractor) writeTextNode(n *html.Node) {
	start := len(x.out)
	last, hasLast := x.findLastByte()
	x.write(n.Data)
	if x.isPlacing && len(x.out) > start {
		x.recordRun(n, start, last, hasLast)
	}
}
