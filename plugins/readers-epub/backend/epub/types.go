package epub

import (
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/markup"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/opf"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/text"
)

// The types a Book is made of are read by their own packages and named here for
// every user of a Book.
type (
	Document  = text.Document
	Part      = text.Part
	Page      = text.Page
	Layout    = opf.Layout
	Direction = opf.Direction
	Markup    = markup.Markup
	Node      = markup.Node
	Attribute = markup.Attribute
)

const (
	Reflowable       = opf.Reflowable
	PrePaginated     = opf.PrePaginated
	DefaultDirection = opf.DefaultDirection
	LeftToRight      = opf.LeftToRight
	RightToLeft      = opf.RightToLeft
	OffsetAttribute  = markup.OffsetAttribute
)
