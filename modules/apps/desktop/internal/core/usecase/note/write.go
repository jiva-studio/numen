package note

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// ErrTooLarge is what a body over MaxBytes gets. Nothing is written. The bound
// is measured against the body being written.
var ErrTooLarge = errors.New("this is more text than a note is written with")

// ErrBodyRefused is a body that opens with the frontmatter delimiter. It is a
// whole note handed back as prose — a caller that read a file, changed it, and
// returned all of it. Writing it would put a second frontmatter block inside
// the first one's note, and the block that then reads as the note's own is the
// wrong one.
var ErrBodyRefused = errors.New(
	"a body is the prose below the frontmatter, and this one begins with a frontmatter block; " +
		"send what note_read gave you, or use the link tools to change the frontmatter")

// ErrUnreadable is a note whose frontmatter is not YAML. A write discovers it
// on the way past and leaves the file alone: repairing the block means guessing
// at what the person wrote.
var ErrUnreadable = markdown.ErrUnreadable

// Write replaces the prose of a note and leaves its frontmatter alone.
//
// The two are separate on purpose. The body is what the person wrote and is
// replaced wholesale; the frontmatter is shared with them, and the keys the
// application does not own survive because nothing here rewrites them. How
// somebody's YAML is formatted stays theirs.
type Write struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	Now     func() time.Time
}

// Execute puts body in the note at path.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrChanged comes
// back: someone editing their own note outranks a caller that read it, thought
// about it, and arrived late.
func (u Write) Execute(ctx context.Context, v domain.Vault, path, body string, fingerprint domain.FileRef) error {
	opening := strings.TrimPrefix(body, "\ufeff")
	if strings.HasPrefix(opening, "---\n") || strings.HasPrefix(opening, "---\r\n") {
		return ErrBodyRefused
	}

	e := editing{
		readers: u.Readers, writers: u.Writers, index: u.Index, now: u.Now,
		fingerprint: fingerprint,
	}
	return e.apply(ctx, v, path, func(doc *markdown.Document) error {
		doc.SetBody(body)
		return nil
	})
}

// Save puts body in the note at path, and makes the note where there is none.
//
// This is the person writing their own note in their own window. There is no
// comparison: the body lands on the frontmatter the file holds at the moment
// the write goes in, and a note carrying no identifier keeps none. A note made
// here is made with no frontmatter at all.
func (u Write) Save(ctx context.Context, v domain.Vault, path, body string) error {
	if len(body) > MaxBytes {
		return fmt.Errorf("%w: %d bytes, and %d is the most", ErrTooLarge, len(body), MaxBytes)
	}
	// A body opening with the delimiter is read back as a frontmatter block, and
	// then the prose it was is no longer the note's body.
	if opening := strings.TrimPrefix(body, "\ufeff"); strings.HasPrefix(opening, "---\n") ||
		strings.HasPrefix(opening, "---\r\n") {
		return ErrBodyRefused
	}
	e := editing{
		readers: u.Readers, writers: u.Writers, index: u.Index,
		overwrite: true,
	}
	return e.apply(ctx, v, path, func(doc *markdown.Document) error {
		doc.SetBody(body)
		return nil
	})
}
