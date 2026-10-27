package domain

import (
	"sort"
)

// Selector locates a segment in the file its text was made from, as JSON text
// (W3C Web Annotation selector). The core stores and returns it as is; the
// plugin SDK documents its keys.
type Selector string

// TextSegment is one segment of a version's text. Heading is the heading the
// segment opens, and empty for a segment inside one.
type TextSegment struct {
	Text     string
	Heading  string
	Selector Selector
}

// VersionText is a version's text as one string, and the segments it was given
// in, each at the byte offset it begins at.
type VersionText struct {
	Version  TextVersion
	Text     string
	segments []TextSegment
	offsets  []int
}

// NewVersionText joins the segments of a version's text in the order given.
func NewVersionText(version TextVersion, segments []TextSegment) VersionText {
	var b []byte
	offsets := make([]int, len(segments))
	for i, s := range segments {
		offsets[i] = len(b)
		b = append(b, s.Text...)
	}
	return VersionText{Version: version, Text: string(b), segments: segments, offsets: offsets}
}

// GetSegments is the segments of the text, in order.
func (t VersionText) GetSegments() []TextSegment { return t.segments }

// GetHeadings is where each heading begins.
func (t VersionText) GetHeadings() []PartStart {
	var out []PartStart
	for i, s := range t.segments {
		if s.Heading != "" {
			out = append(out, PartStart{Title: s.Heading, Offset: t.offsets[i]})
		}
	}
	return out
}

// GetHeadingsAt is the headings that begin exactly at an offset, in the order
// their segments were given.
func (t VersionText) GetHeadingsAt(offset int) []string {
	var names []string
	for i, s := range t.segments {
		if t.offsets[i] == offset && s.Heading != "" {
			names = append(names, s.Heading)
		}
	}
	return names
}

// Locate is the heading an offset falls under, and empty where none precedes it.
func (t VersionText) Locate(offset int) string {
	for j := t.findSegment(offset); j >= 0; j-- {
		if t.segments[j].Heading != "" {
			return t.segments[j].Heading
		}
	}
	return ""
}

// GetSelector is the selector of the segment an offset falls in, and empty
// where the offset is before the first segment or the segment has none.
func (t VersionText) GetSelector(offset int) Selector {
	i := t.findSegment(offset)
	if i < 0 {
		return ""
	}
	return t.segments[i].Selector
}

// findSegment is the index of the last segment that begins at or before an
// offset, and -1 where none does.
func (t VersionText) findSegment(offset int) int {
	return sort.Search(len(t.offsets), func(i int) bool { return t.offsets[i] > offset }) - 1
}
