// Package scan holds what the recognizer reads off a page drawn as an image: its regions, the boxes
// of each region and the models it read them with.
package scan

import "image"

// Model describes active OCR models.
type Model struct {
	Layout     string
	Recognizer string
	DPI        int
	From       string
}

// ByteRange represents a byte offset range.
type ByteRange struct {
	Start int
	End   int
}

// Box is one box of a region: the rectangle it covers on the page, and the run of bytes it produced.
type Box struct {
	Rect  image.Rectangle
	Range ByteRange
}

// Block is one region of a page, written out.
type Block struct {
	Label     string
	Text      string
	IsHeading bool
	Depth     int
	Boxes     []Box
}
