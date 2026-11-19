package domain

import "cmp"

// FileType identifies the content type of a file and the frontmatter type of an indexed note.
type FileType struct {
	Content ContentType
	Note    NoteType
}

// CompareFileTypes sorts by content type, then by note type.
func CompareFileTypes(a, b FileType) int {
	return cmp.Or(cmp.Compare(a.Content, b.Content), cmp.Compare(a.Note, b.Note))
}
