package mcp

import (
	"fmt"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/refusal"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// What a tool says about a refusal. Which refusal a thing is comes from
// refusal, the same as it does for a window; the sentence is this one's own,
// because an agent is told which tool to reach for next and a person is not.

// why is a read's outcome in words an agent can act on.
func why(c note.Contents) string {
	reason, refused := refusal.Of(c.Outcome)
	if !refused {
		return string(c.Outcome)
	}
	// A note past the bound is answered with the size it came to: that is what
	// an agent asks the file tools for a stretch of.
	if reason == v1.Refusal_REFUSAL_TOO_LARGE {
		return fmt.Sprintf("it is %d bytes, larger than the %d this reads; open the file instead",
			c.Ref.Size, note.MaxBytes)
	}
	return said(reason)
}

// refusing is a write's error in words an agent can act on. An error that is no
// refusal is handed over as it stands.
func refusing(err error) string {
	if reason, refused := refusal.By(err); refused {
		return said(reason)
	}
	return err.Error()
}

// said is one refusal in the words a tool answers with.
func said(reason v1.Refusal) string {
	switch reason {
	case v1.Refusal_REFUSAL_MISSING:
		return "the vault holds no note at this path"
	case v1.Refusal_REFUSAL_NOT_A_NOTE:
		return "this is not a note the vault holds"
	case v1.Refusal_REFUSAL_NOT_TEXT:
		return "this file is not text: some of it is not valid UTF-8, so open it as a file"
	case v1.Refusal_REFUSAL_TOO_LARGE:
		return fmt.Sprintf("it is larger than the %d bytes this reads; open the file instead",
			note.MaxBytes)
	case v1.Refusal_REFUSAL_BODY_REFUSED:
		return "this text opens with a frontmatter delimiter, so it cannot be written into a note"
	case v1.Refusal_REFUSAL_UNREADABLE:
		return "the frontmatter of this note cannot be read, so it can be neither read nor written from here"
	case v1.Refusal_REFUSAL_OCCUPIED:
		return "a file is already where this note would go"
	case v1.Refusal_REFUSAL_UNNAMEABLE:
		return "a note cannot be called this"
	case v1.Refusal_REFUSAL_NOT_A_STENCIL:
		return "this note is not a stencil"
	case v1.Refusal_REFUSAL_NOT_A_DECK:
		return "this note is not a deck"
	case v1.Refusal_REFUSAL_DECK_TOO_LARGE:
		return "this deck is larger than a deck is read at"
	case v1.Refusal_REFUSAL_NOT_A_PRESET:
		return "this note is not a preset"
	}
	return reason.String()
}
