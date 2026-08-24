package note

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// Naming is which of the three a note is shown by, and so which one naming it
// writes.
type Naming string

const (
	ByFrontmatter Naming = "frontmatter"
	ByHeading     Naming = "heading"
	ByFilename    Naming = "filename"
)

// ErrUnnameable is a title no note can be given. Nothing is written.
var ErrUnnameable = errors.New("a note cannot be given this title")

// ErrNotAHeading is a title a level-one heading is read back as something else.
// A note the heading names is refused it, and a note the `title` key names
// carries it there.
var ErrNotAHeading = markdown.ErrNotAHeading

// nameOf is the filename a title is filed under, and whether the filename is
// the whole of the title.
//
// Two titles are refused whatever the note. One that leaves no filename names
// nothing. One carrying a line break is a name no list, tab or heading draws on
// one line, and the filename joins its words without saying so.
//
// The title comes in trimmed.
func nameOf(title string) (name string, exact bool, err error) {
	switch name, exact = domain.Filename(title); {
	case name == "":
		return "", false, fmt.Errorf("%w: %q leaves nothing a file can be named after", ErrUnnameable, title)
	case strings.ContainsAny(title, "\n\r"):
		return "", false, fmt.Errorf("%w: %q is more than one line", ErrUnnameable, title)
	}
	return name, exact, nil
}
