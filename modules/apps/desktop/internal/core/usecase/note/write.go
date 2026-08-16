package note

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

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
	// A body that opens with the delimiter is a whole note being handed back as
	// prose — a caller that read a file, changed it, and returned all of it.
	// Writing it would put a second frontmatter block inside the first one's
	// note, and the block that then reads as the note's own is the wrong one.
	opening := strings.TrimPrefix(body, "\ufeff")
	if strings.HasPrefix(opening, "---\n") || strings.HasPrefix(opening, "---\r\n") {
		return errors.New("a body is the prose below the frontmatter, and this one begins with a frontmatter block; " +
			"send what note_read gave you, or use the link tools to change the frontmatter")
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
