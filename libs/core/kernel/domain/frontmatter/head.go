package frontmatter

import "github.com/jiva-studio/numen/libs/shared/note/domain"

// Head is what the index holds of a note that decides the tab it opens in: its type and its
// frontmatter. Values is nil where the note has no frontmatter, or none that could be read.
type Head struct {
	Type   domain.NoteType
	Values map[string]any
}
