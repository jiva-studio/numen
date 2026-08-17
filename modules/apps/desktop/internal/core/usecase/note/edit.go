package note

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
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
	// overwrite is a caller writing what is in front of the person: the note
	// on disk is replaced without being compared to anything, a note that is
	// not there is made, and no identifier is stamped.
	overwrite bool
}

func (e editing) apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) error {
	if err := e.splice(ctx, v, path, change); err != nil {
		return err
	}
	if e.index == nil {
		return nil
	}
	return e.index(ctx, v, []string{path})
}

// splice is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (e editing) splice(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) error {
	release, err := e.writers.Hold(ctx, v)
	if err != nil {
		return err
	}
	defer release()

	reader, err := e.readers.Open(v)
	if err != nil {
		return err
	}
	raw, err := reader.Read(ctx, path)
	switch {
	case err == nil:
	case e.overwrite && errors.Is(err, fs.ErrNotExist):
		// The note is made by this write, out of the person's own text and
		// nothing else: no frontmatter, and no identifier. An identifier
		// arrives when the application changes a note's contents.
		raw = nil
	default:
		return fmt.Errorf("read %s: %w", path, err)
	}

	// Every edit is a read, a think and a write, and the person may save the
	// note in their own editor in between. A caller that said what it believed
	// the note was is held to that; one that said nothing is held to what was
	// read just now. A caller writing over what is there is held to neither.
	against := e.fingerprint
	if against == (domain.FileRef{}) && !e.overwrite {
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
	// A caller writing over what is there is the person editing their own note,
	// and leaves the frontmatter as they wrote it.
	if _, carried := doc.Identifier(); !carried && !e.overwrite {
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
	return writer.Write(ctx, path, doc.Bytes(), against)
}
