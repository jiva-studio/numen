package cards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// Write puts a deck or a stencil back where it came from.
//
// A deck and a stencil are notes, so this is a note's write with a deck's bound
// on it: the body is replaced whole, and a file that has changed since the
// caller read it is left alone with port.ErrChanged. The frontmatter is the
// person's, apart from the one key a stencil declares its fields in.
type Write struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Now is when this is happening. An identifier written here carries it.
	Now func() time.Time
}

// Deck puts body in the deck at path, and answers with the fingerprint the
// write produced, which is what the caller presents at its next write.
func (u Write) Deck(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	if len(body) > MaxBytes {
		return domain.FileRef{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a deck is", note.ErrTooLarge, len(body), MaxBytes)
	}
	return u.note().Execute(ctx, v, path, body, fingerprint)
}

// Stencil puts the faces and the fields into the stencil at path. A stencil is
// a note and is bounded as one.
//
// The faces are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both.
func (u Write) Stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.FileRef,
) (domain.FileRef, error) {
	if len(body) > note.MaxBytes {
		return domain.FileRef{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a stencil is", note.ErrTooLarge, len(body), note.MaxBytes)
	}
	at, err := u.stencil(ctx, v, path, body, fields, fingerprint)
	if err != nil || u.Index == nil {
		return at, err
	}
	return at, u.Index(ctx, v, []string{path})
}

// stencil is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (u Write) stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.FileRef,
) (domain.FileRef, error) {
	// A body opening with the delimiter is read back as a frontmatter block, and
	// then the prose it was is no longer the note's body. A byte order mark —
	// "\xef\xbb\xbf" — stands before the delimiter and is no part of the prose.
	if opening := strings.TrimPrefix(body, "\xef\xbb\xbf"); strings.HasPrefix(opening, "---\n") ||
		strings.HasPrefix(opening, "---\r\n") {
		return domain.FileRef{}, note.ErrBodyRefused
	}

	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return domain.FileRef{}, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}
	// A caller that said what it believed the stencil was is held to that; one
	// that said nothing is held to what stands there now.
	against := fingerprint
	if against == (domain.FileRef{}) {
		on, err := reader.Stat(ctx, path)
		if err != nil {
			return domain.FileRef{}, fmt.Errorf("look at %s: %w", path, missing(err))
		}
		against = on
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("read %s: %w", path, missing(err))
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
	}

	doc.SetBody(body)
	// A stencil already declaring these, in this order, keeps the bytes the
	// person wrote them as.
	if declared, _ := doc.List(fieldsKey); !slices.Equal(declared, fields) {
		if err := doc.SetList(fieldsKey, fields); err != nil {
			return domain.FileRef{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	// The application is changing what this note holds, so it writes the
	// identifier the note does not carry.
	if _, carried := doc.Identifier(); !carried {
		identifier, err := ulid.New(u.now())
		if err != nil {
			return domain.FileRef{}, err
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return domain.FileRef{}, err
		}
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return domain.FileRef{}, err
	}
	return writer.Write(ctx, path, doc.Bytes(), against)
}

// missing is note.ErrNoNote where the vault holds nothing at the path, and the
// error as it arrived otherwise.
func missing(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return note.ErrNoNote
	}
	return err
}

func (u Write) now() time.Time {
	if u.Now == nil {
		return time.Now()
	}
	return u.Now()
}

func (u Write) note() note.Write {
	return note.Write{Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now}
}
