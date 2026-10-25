package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TextPlace is where a part of a text lies in the file it was made from: on
// a page and inside rectangles of it, for positioned text; between two moments,
// for timed text. Plain text has no place.
//
// Page counts from zero. Rects are fractions of the page.
type TextPlace struct {
	Page  int
	Rects []Rect
	Start time.Duration
	End   time.Duration
}

// TextPart is one run of a version's text and its place. Section names
// the section the part opens, and is empty for a part inside one.
type TextPart struct {
	Text    string
	Place   TextPlace
	Section string
}

// VersionText is a version's text as one string, and the parts it was given
// in, each at the byte offset it begins at.
type VersionText struct {
	Version TextVersion
	Text    string
	parts   []TextPart
	offsets []int
}

// NewVersionText joins the parts of a version's text in the order given.
func NewVersionText(version TextVersion, parts []TextPart) VersionText {
	var b strings.Builder
	offsets := make([]int, len(parts))
	for i, p := range parts {
		offsets[i] = b.Len()
		b.WriteString(p.Text)
	}
	return VersionText{Version: version, Text: b.String(), parts: parts, offsets: offsets}
}

// GetSections is where each named section begins.
func (t VersionText) GetSections() []PartStart {
	var out []PartStart
	for i, p := range t.parts {
		if p.Section != "" {
			out = append(out, PartStart{Title: p.Section, Offset: t.offsets[i]})
		}
	}
	return out
}

// GetNamesAt is the names of the sections that begin exactly at an offset, in
// the order their parts were given.
func (t VersionText) GetNamesAt(offset int) []string {
	var names []string
	for i, p := range t.parts {
		if t.offsets[i] == offset && p.Section != "" {
			names = append(names, p.Section)
		}
	}
	return names
}

// Locate describes an offset: the section it falls in, and the page or the
// moment of its part. Empty where the version names
// neither.
func (t VersionText) Locate(offset int) string {
	i := sort.Search(len(t.offsets), func(i int) bool { return t.offsets[i] > offset }) - 1
	if i < 0 {
		return ""
	}
	var named []string
	for j := i; j >= 0; j-- {
		if t.parts[j].Section != "" {
			named = append(named, t.parts[j].Section)
			break
		}
	}
	switch t.Version.Format {
	case TextFormatPositioned:
		named = append(named, fmt.Sprintf("page %d of the file", t.parts[i].Place.Page+1))
	case TextFormatTimed:
		named = append(named, FormatClock(t.parts[i].Place.Start))
	case TextFormatPlain:
	}
	return strings.Join(named, ", ")
}

// GetBoxes is the rectangles of the positioned parts a run of the text
// overlaps, each with the run of the part it covers.
func (t VersionText) GetBoxes(run ByteSpan) []HighlightBox {
	if t.Version.Format != TextFormatPositioned || run.Empty() {
		return nil
	}
	var boxes []HighlightBox
	for i, p := range t.parts {
		from, to := t.offsets[i], t.offsets[i]+len(p.Text)
		if to <= run.From || from >= run.To {
			continue
		}
		for _, rect := range p.Place.Rects {
			boxes = append(boxes, HighlightBox{Page: p.Place.Page, ByteSpan: ByteSpan{From: from, To: to}, Rect: rect})
		}
	}
	return boxes
}

// FormatClock is a moment of a recording as a person reads it: minutes and
// seconds, with hours in front once there are any.
func FormatClock(offset time.Duration) string {
	whole := max(int64(offset/time.Second), 0)
	m, s := whole/60%60, whole%60
	if hours := whole / 3600; hours > 0 {
		return strconv.FormatInt(hours, 10) + fmt.Sprintf(":%02d:%02d", m, s)
	}
	return strconv.FormatInt(whole/60, 10) + fmt.Sprintf(":%02d", s)
}
