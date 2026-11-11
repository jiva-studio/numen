package text

import (
	"bytes"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/words"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A frame tracks an open element during tokenization.
type frame struct {
	atom        atom.Atom
	start       int
	block       bool
	heading     int
	isPagebreak bool
	pageLabel   string
}

type tokenizer struct {
	x                  *Extractor
	z                  *html.Tokenizer
	stack              []frame
	skipAtom           atom.Atom
	dropLeadingNewline bool
}

type tagAttrs struct {
	id          string
	isPagebreak bool
	pageLabel   string
}

func (t *tokenizer) parseTagAttrs(hasAttr bool) tagAttrs {
	var attrs tagAttrs
	var titleAttr, ariaLabelAttr string
	for hasAttr {
		var key, val []byte
		key, val, hasAttr = t.z.TagAttr()
		switch {
		case bytes.Equal(key, []byte("id")):
			attrs.id = string(val)
		case bytes.Equal(key, []byte("epub:type")):
			if words.HasTokenBytes(val, "pagebreak") {
				attrs.isPagebreak = true
			}
		case bytes.Equal(key, []byte("role")):
			if words.HasTokenBytes(val, "doc-pagebreak") {
				attrs.isPagebreak = true
			}
		case bytes.Equal(key, []byte("title")):
			titleAttr = string(val)
		case bytes.Equal(key, []byte("aria-label")):
			ariaLabelAttr = string(val)
		}
	}
	if attrs.isPagebreak {
		if s := words.NormalizeSpaces(titleAttr); s != "" {
			attrs.pageLabel = s
		} else if al := words.NormalizeSpaces(ariaLabelAttr); al != "" {
			attrs.pageLabel = al
		}
	}
	return attrs
}

func (t *tokenizer) handleFinish() {
	for _, v := range slices.Backward(t.stack) {
		t.x.closeFrame(v)
	}
	t.x.preformattedDepth = 0
}

func (t *tokenizer) handleText() {
	if t.skipAtom == 0 {
		text := t.z.Text()
		if t.dropLeadingNewline {
			text = bytes.TrimPrefix(bytes.TrimPrefix(text, []byte{'\r'}), []byte{'\n'})
		}
		t.x.writeBytes(text)
	}
	t.dropLeadingNewline = false
}

func (t *tokenizer) shouldSkipStartTag(tt html.TokenType, a atom.Atom) bool {
	if t.skipAtom != 0 {
		if isBlock(a) {
			t.skipAtom = 0
		} else {
			return true
		}
	}
	if a == atom.Head || a == atom.Script || a == atom.Style || a == atom.Title {
		if tt == html.StartTagToken {
			t.skipAtom = a
		}
		return true
	}
	return false
}

func (t *tokenizer) adjustElementState(a atom.Atom) {
	if a == atom.Pre {
		t.x.preformattedDepth++
		t.dropLeadingNewline = true
	}
	if a == atom.Listing || a == atom.Textarea {
		t.dropLeadingNewline = true
	}
	if a == atom.Td || a == atom.Th {
		t.x.writeBytes([]byte(" "))
	}
}

func (t *tokenizer) handleStartTag(tt html.TokenType) {
	t.dropLeadingNewline = false
	name, hasAttr := t.z.TagName()
	a := atom.Lookup(name)

	if t.shouldSkipStartTag(tt, a) {
		return
	}
	if a == atom.Br {
		t.x.breakLine()
		return
	}

	block := isBlock(a)
	if block {
		t.x.breakLine()
	}

	attrs := t.parseTagAttrs(hasAttr)
	if attrs.id != "" {
		t.x.anchors[t.x.path+"#"+attrs.id] = len(t.x.out)
	}

	start := len(t.x.out)
	t.adjustElementState(a)

	level := parseHeadingLevel(a)
	if isVoidTag(a) {
		t.x.closeFrame(frame{atom: 0, start: start, block: false, heading: 0, isPagebreak: attrs.isPagebreak, pageLabel: attrs.pageLabel})
		// gomutants:disable-next-line BRANCH_IF reason="the preceding line break leaves the buffer ending in a newline"
		if block {
			t.x.breakLine()
		}
		return
	}

	t.stack = append(t.stack, frame{
		atom:        a,
		start:       start,
		block:       block,
		heading:     level,
		isPagebreak: attrs.isPagebreak,
		pageLabel:   attrs.pageLabel,
	})
}

func (t *tokenizer) findStackFrame(a atom.Atom) int {
	for i, v := range slices.Backward(t.stack) {
		if v.atom == a {
			return i
		}
	}
	return -1
}

func (t *tokenizer) popFramesUntil(target int) {
	for i := len(t.stack) - 1; i >= target; i-- {
		t.x.closeFrame(t.stack[i])
		if t.stack[i].block {
			t.x.breakLine()
		}
	}
	t.stack = t.stack[:target]
}

func (t *tokenizer) handleEndTag() {
	name, _ := t.z.TagName()
	a := atom.Lookup(name)

	if t.skipAtom != 0 {
		if a == t.skipAtom {
			t.skipAtom = 0
		}
		return
	}

	if a == atom.Br || a == atom.Body || a == atom.Html {
		if a == atom.Br {
			t.x.breakLine()
		}
		return
	}

	found := t.findStackFrame(a)
	if found >= 0 {
		t.popFramesUntil(found)
	} else if isBlock(a) {
		t.x.breakLine()
	}
}

func (x *Extractor) tokenize(raw []byte) {
	t := tokenizer{
		x: x,
		z: html.NewTokenizer(bytes.NewReader(raw)),
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="the capacity does not change the slice's contents"
		stack:              make([]frame, 0, 32),
		skipAtom:           0,
		dropLeadingNewline: false,
	}

	for {
		tt := t.z.Next()
		switch tt {
		case html.ErrorToken:
			t.handleFinish()
			return
		case html.CommentToken, html.DoctypeToken:
		case html.TextToken:
			t.handleText()
		case html.StartTagToken, html.SelfClosingTagToken:
			t.handleStartTag(tt)
		case html.EndTagToken:
			t.handleEndTag()
		}
	}
}

func (x *Extractor) closeFrame(f frame) {
	if f.atom == atom.Pre {
		x.preformattedDepth--
	}
	if f.heading > 0 {
		if title := words.NormalizeSpaces(string(x.out[f.start:])); title != "" {
			x.headings = append(x.headings, Part{Title: title, Offset: f.start, Level: f.heading})
		}
	}
	if f.isPagebreak {
		label := f.pageLabel
		if label == "" {
			label = words.NormalizeSpaces(string(x.out[f.start:]))
		}
		if label != "" {
			x.pagebreaks = append(x.pagebreaks, Page{Label: label, Offset: f.start})
		}
	}
}

func isVoidTag(key atom.Atom) bool {
	switch key {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Embed, atom.Hr, atom.Img, atom.Input,
		atom.Keygen, atom.Link, atom.Meta, atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}
	return false
}

func isSkipped(key atom.Atom) bool {
	switch key {
	case atom.Head, atom.Script, atom.Style, atom.Title:
		return true
	}
	return false
}

// GetAttribute reads an attribute of the element by name. The markup is read as
// HTML, where a prefix is part of the name: `epub:type` is an attribute called
// "epub:type".
func GetAttribute(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, name) {
			return a.Val
		}
	}
	return ""
}

func parseHeadingLevel(a atom.Atom) int {
	switch a {
	case atom.H1:
		return 1
	case atom.H2:
		return 2
	case atom.H3:
		return 3
	case atom.H4:
		return 4
	case atom.H5:
		return 5
	case atom.H6:
		return 6
	}
	return 0
}

func isBlock(key atom.Atom) bool {
	switch key {
	case atom.Address, atom.Article, atom.Aside, atom.Blockquote, atom.Body, atom.Caption, atom.Dd,
		atom.Div, atom.Dl, atom.Dt, atom.Figcaption, atom.Figure, atom.Footer, atom.H1, atom.H2, atom.H3,
		atom.H4, atom.H5, atom.H6, atom.Header, atom.Hr, atom.Li, atom.Main, atom.Nav, atom.Ol, atom.P,
		atom.Pre, atom.Section, atom.Table, atom.Tr, atom.Ul:
		return true
	}
	return false
}
