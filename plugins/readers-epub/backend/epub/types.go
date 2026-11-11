package epub

import (
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/markup"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
)

// The types a Book is made of are read by their own packages and named here for
// every user of a Book.
type (
	// Document is a spine document and where it begins in the book's Text.
	Document = text.Document
	// Part is a named part of the book.
	Part = text.Part
	// Page is a page of the printed book.
	Page = text.Page
	// Layout is how a book's pages are drawn.
	Layout = opf.Layout
	// Direction is the direction a book's pages progress in.
	Direction = opf.Direction
	// Markup is a document as the markup a window draws.
	Markup = markup.Markup
	// Node is one element of a document, or one run of its text.
	Node = markup.Node
	// Attribute is one attribute of an element.
	Attribute = markup.Attribute
)

// The layouts, directions and markup attribute a Book reports.
const (
	Reflowable       = opf.Reflowable
	PrePaginated     = opf.PrePaginated
	DefaultDirection = opf.DefaultDirection
	LeftToRight      = opf.LeftToRight
	RightToLeft      = opf.RightToLeft
	OffsetAttribute  = markup.OffsetAttribute
)
