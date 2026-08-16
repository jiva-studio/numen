package note

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// editing is what every change to the contents of an existing note needs.
//
// It is one function because the shape is always the same and the rules in it
// are easy to forget one at a time: read, refuse what cannot be read, write the
// identifier because this is an edit, change the one thing, put it back, and
// bring the index level.
type editing struct {
	readers     port.VaultReaders
	writers     port.VaultWriters
	index       func(ctx context.Context, v domain.Vault, paths []string) error
	now         func() time.Time
	fingerprint domain.FileRef
}

func (e editing) apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) error {
	reader, err := e.readers.Open(v)
	if err != nil {
		return err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Every edit is a read, a think and a write, and the person may save the
	// note in their own editor in between. A caller that said what it believed
	// the note was is held to that; one that said nothing is held to what was
	// read just now, so no edit here can land on top of theirs unseen.
	against := e.fingerprint
	if against == (domain.FileRef{}) {
		if against, err = reader.Stat(ctx, path); err != nil {
			return fmt.Errorf("look at %s: %w", path, err)
		}
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	if err := change(doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	// The application is editing this note, so it may write the identifier the
	// note does not have. Writing one is what editing a note permits and reading
	// past it does not: nothing backfills an identifier into a note it only read.
	if _, carried := doc.Identifier(); !carried {
		at := time.Now
		if e.now != nil {
			at = e.now
		}
		identifier, err := ulid.New(at())
		if err != nil {
			return err
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return err
		}
	}

	writer, err := e.writers.Open(v)
	if err != nil {
		return err
	}
	if err := writer.Write(ctx, path, doc.Bytes(), against); err != nil {
		return err
	}
	if e.index == nil {
		return nil
	}
	return e.index(ctx, v, []string{path})
}
