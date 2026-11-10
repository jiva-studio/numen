package plugin

import "github.com/jiva-studio/numen/libs/note/domain"

// TextPut is what a plugin reports of one of its text versions. File is
// the name of the version's file in the plugin's folder.
type TextPut struct {
	Path           string
	Key            string
	Source         domain.TextVersionID
	SourceProperty domain.TextSourceProperty
	Kind           domain.TextKind
	InputHash      string
	File           string
}

// TextListing is the versions of a file and the live one, which is empty
// where none is live.
type TextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// TextPage is one page of a version's segments. An empty NextCursor is the last page.
type TextPage struct {
	Segments   []domain.TextSegment
	NextCursor string
}

// TextRange is a ranged read of a version: the segments, the start of the first in the whole
// text, and the size of the whole text in bytes.
type TextRange struct {
	Segments []domain.TextSegment
	Start    int
	Size     int
}
