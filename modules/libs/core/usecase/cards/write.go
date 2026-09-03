package cards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"time"

	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/mark"
	"github.com/jiva-studio/numen/modules/libs/core/internal/ulid"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
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
	// Links answers where the wikilink a card names its stencil by lands, which
	// is what says which of the card's fields is first. A build holding none
	// reprojects no heading.
	Links port.LinkQueries
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	// Now is when this is happening. An identifier written here carries it.
	Now func() time.Time
}

// Wrote is what a write of a deck left behind: what the file now stands at,
// which is what the caller presents at its next write, and the mark every card
// that carried none was given.
type Wrote struct {
	At     domain.Fingerprint
	Minted []format.Minted
}

// Deck puts body in the deck at path.
//
// The deck is made whole on the way past: every card carrying no mark is given
// one, and every stale heading is put back in step. Every caller therefore gets
// the same file, and none of them has to know the rules.
func (u Write) Deck(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.Fingerprint,
) (Wrote, error) {
	if len(body) > MaxBytes {
		return Wrote{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a deck is", note.ErrTooLarge, len(body), MaxBytes)
	}
	whole, minted, err := u.whole(ctx, v, path, body)
	if err != nil {
		return Wrote{}, err
	}
	at, err := u.note().Execute(ctx, v, path, whole, fingerprint)
	if err != nil {
		return Wrote{}, err
	}
	return Wrote{At: at, Minted: minted}, u.level(ctx, v, path)
}

// whole is the body every card of which has been made whole, and the marks that
// took. The stencils are read against the body being written, so a card whose
// wikilink the caller has just changed is cut by the stencil it now names.
func (u Write) whole(
	ctx context.Context, v domain.Vault, path, body string,
) (string, []format.Minted, error) {
	read := Read{Readers: u.Readers, Links: u.Links}
	by, err := read.Cutting(ctx, v, path, format.ReadDeck(domain.Note{Body: body}))
	if err != nil {
		return "", nil, err
	}
	whole, minted, err := format.Whole(body, by, mark.New)
	if err != nil {
		return "", nil, fmt.Errorf("make %s whole: %w", path, err)
	}
	return whole, minted, nil
}

// Stencil puts the faces and the fields into the stencil at path. A stencil is
// a note and is bounded as one.
//
// The faces are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both.
func (u Write) Stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if len(body) > note.MaxBytes {
		return domain.Fingerprint{}, fmt.Errorf(
			"%w: %d bytes, and %d is the most a stencil is", note.ErrTooLarge, len(body), note.MaxBytes)
	}
	at, err := u.stencil(ctx, v, path, body, fields, fingerprint)
	if err != nil {
		return at, err
	}
	return at, u.level(ctx, v, path)
}

// stencil is the read, the change and the write, under this vault's write lock
// from before the read until after the file is replaced.
func (u Write) stencil(
	ctx context.Context, v domain.Vault, path, body string, fields []string,
	fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	// A body opening with the delimiter is read back as a frontmatter block, and
	// then the prose it was is no longer the note's body. A byte order mark —
	// "\xef\xbb\xbf" — stands before the delimiter and is no part of the prose.
	if opening := strings.TrimPrefix(body, "\xef\xbb\xbf"); strings.HasPrefix(opening, "---\n") ||
		strings.HasPrefix(opening, "---\r\n") {
		return domain.Fingerprint{}, note.ErrBodyRefused
	}

	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	// A caller that said what it believed the stencil was is held to that; one
	// that said nothing is held to what stands there now.
	against := fingerprint
	if against == (domain.Fingerprint{}) {
		on, err := reader.Stat(ctx, path)
		if err != nil {
			return domain.Fingerprint{}, fmt.Errorf("look at %s: %w", path, missing(err))
		}
		against = on
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("read %s: %w", path, missing(err))
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%s: %w", path, err)
	}

	doc.SetBody(body)
	// A stencil already declaring these, in this order, keeps the bytes the
	// person wrote them as.
	if declared, _ := doc.List(fieldsKey); !slices.Equal(declared, fields) {
		if err := doc.SetList(fieldsKey, fields); err != nil {
			return domain.Fingerprint{}, fmt.Errorf("%s: %w", path, err)
		}
	}
	// The application is changing what this note holds, so it writes the
	// identifier the note does not carry.
	if _, carried := doc.Identifier(); !carried {
		identifier, err := ulid.New(u.now())
		if err != nil {
			return domain.Fingerprint{}, err
		}
		if err := doc.SetIdentifier(identifier); err != nil {
			return domain.Fingerprint{}, err
		}
	}

	writer, err := u.Writers.Open(v)
	if err != nil {
		return domain.Fingerprint{}, err
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

// level brings what a write touched up to date. The levelling is done here so
// that a caller holding the file's fingerprint is told which of the two failed.
func (u Write) level(ctx context.Context, v domain.Vault, path string) error {
	if u.Index == nil {
		return nil
	}
	return note.Levelled(path, u.Index(ctx, v, []string{path}))
}

// note is the writer a deck goes to disk through, held to the size a deck is
// read at.
func (u Write) note() note.Write {
	return note.Write{Readers: u.Readers, Writers: u.Writers, Now: u.Now, Bound: MaxBytes}
}
