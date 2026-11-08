// Package cfi parses and writes EPUB Canonical Fragment Identifiers (IDPF EPUB CFI 1.1) of the form
// the reader uses: the steps through the package document to a spine item, an indirection, the
// steps through that item's document to a node, and a character offset inside a text node.
package cfi

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ErrSyntax is a string that is no CFI the reader can follow.
var ErrSyntax = errors.New("cfi: invalid syntax")

// A CFI is a place in an EPUB publication.
type CFI struct {
	Package  []int
	Document []int
	Offset   int
}

// String is the CFI as the spec writes it, such as "epubcfi(/6/4!/4/2/1:12)".
func (c CFI) String() string {
	var b strings.Builder
	b.WriteString("epubcfi(")
	writeSteps(&b, c.Package)
	b.WriteByte('!')
	writeSteps(&b, c.Document)
	if c.Offset >= 0 {
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(c.Offset))
	}
	b.WriteByte(')')
	return b.String()
}

func writeSteps(b *strings.Builder, steps []int) {
	for _, step := range steps {
		b.WriteByte('/')
		b.WriteString(strconv.Itoa(step))
	}
}

// Parse reads a CFI. ID assertions and side bias are skipped, a range CFI is read as its start,
// and a spatial or temporal offset is dropped.
func Parse(s string) (CFI, error) {
	inner, ok := strings.CutPrefix(s, "epubcfi(")
	if !ok || !strings.HasSuffix(inner, ")") {
		return CFI{}, fmt.Errorf("%w: %q has no epubcfi( )", ErrSyntax, s)
	}
	inner = inner[:len(inner)-1]
	p := parser{in: inner, pos: 0}
	parent, err := p.readPath()
	if err != nil {
		return CFI{}, fmt.Errorf("%w in %q", err, s)
	}
	parent, err = p.readEnd(parent, s)
	if err != nil {
		return CFI{}, err
	}
	if len(parent.indirected) != 2 || len(parent.steps) == 0 && parent.offset >= 0 {
		return CFI{}, fmt.Errorf("%w: %q does not step through the spine into a document", ErrSyntax, s)
	}
	return CFI{Package: parent.indirected, Document: parent.steps, Offset: parent.offset}, nil
}

func (p *parser) readEnd(parent path, s string) (path, error) {
	if p.peek() != ',' {
		if p.pos != len(p.in) {
			// gomutants:disable-next-line RETURN_ZERO reason="parent is ignored when readEnd returns an error"
			return parent, fmt.Errorf("%w: %q has %q after its path", ErrSyntax, s, p.in[p.pos:])
		}
		return parent, nil
	}
	// A range is the parent path followed by the start's local path.
	p.pos++
	start, err := p.readPath()
	if err != nil {
		return parent, fmt.Errorf("%w in %q", err, s)
	}
	parent = mergeRange(parent, start)
	if p.peek() != ',' {
		// gomutants:disable-next-line RETURN_ZERO reason="parent is ignored when readEnd returns an error"
		return parent, fmt.Errorf("%w: %q has a range with no end", ErrSyntax, s)
	}
	return parent, nil
}

// path is a CFI path as read: the steps before the indirection, the steps after it, and the offset.
type path struct {
	indirected []int
	steps      []int
	hasBang    bool
	offset     int
}

func mergeRange(parent, start path) path {
	if start.hasBang {
		parent.indirected = slices.Concat(parent.steps, start.indirected)
		parent.steps = start.steps
		// gomutants:disable-next-line STATEMENT_REMOVE reason="hasBang tracks indirection presence on merged range path"
		parent.hasBang = true
	} else {
		parent.steps = append(parent.steps, start.steps...)
	}
	parent.offset = start.offset
	return parent
}

type parser struct {
	in  string
	pos int
}

func (p *parser) peek() byte {
	if p.pos >= len(p.in) {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="any non-delimiter sentinel byte indicates end of input"
		return 0
	}
	return p.in[p.pos]
}

func (p *parser) readPath() (path, error) {
	read := path{indirected: nil, steps: nil, hasBang: false, offset: -1}
	for {
		switch p.peek() {
		case '/':
			p.pos++
			step, err := p.readInteger()
			if err != nil {
				return path{}, err
			}
			p.skipAssertion()
			read.steps = append(read.steps, step)
		case '!':
			if read.hasBang {
				return path{}, fmt.Errorf("%w: a second indirection", ErrSyntax)
			}
			p.pos++
			read.hasBang = true
			read.indirected, read.steps = read.steps, nil
		case ':':
			p.pos++
			offset, err := p.readInteger()
			if err != nil {
				return path{}, err
			}
			read.offset = offset
			p.skipAssertion()
			p.skipSpatial()
			return read, nil
		default:
			// A temporal offset begins with ~ and a spatial one with @.
			if strings.IndexByte("~@", p.peek()) >= 0 {
				p.skipSpatial()
			}
			return read, nil
		}
	}
}

func (p *parser) readInteger() (int, error) {
	start := p.pos
	for p.pos < len(p.in) && p.in[p.pos] >= '0' && p.in[p.pos] <= '9' {
		p.pos++
	}
	if p.pos == start {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="zero integer is ignored when readInteger returns an error"
		return 0, fmt.Errorf("%w: a step or offset without a number", ErrSyntax)
	}
	return strconv.Atoi(p.in[start:p.pos])
}

func (p *parser) skipAssertion() {
	if p.peek() != '[' {
		return
	}
	// gomutants:disable-next-line STATEMENT_REMOVE,INCREMENT_DECREMENT,CONDITIONALS_BOUNDARY reason="opening bracket is non-^ and non-], so first loop iteration advances past it"
	for p.pos++; p.pos < len(p.in); p.pos++ {
		switch p.in[p.pos] {
		case '^':
			p.pos++
		case ']':
			p.pos++
			return
		}
	}
}

func (p *parser) skipSpatial() {
	for p.pos < len(p.in) && p.in[p.pos] != ',' {
		if p.in[p.pos] == '[' {
			p.skipAssertion()
			continue
		}
		p.pos++
	}
}
