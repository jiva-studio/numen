package cards

import (
	"context"
	"fmt"
	"time"

	format "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/cards"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/mark"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/ulid"
)

// Field is which field of which stencil is being renamed, and to what.
type Field struct {
	// Stencil is where the stencil is filed.
	Stencil string
	From    string
	To      string
	// At is the stencil as the caller read it. A stencil that has changed since
	// is left alone, and no deck is written.
	At domain.FileRef
}

// NotWritten is one deck a rename did not reach. It keeps the old heading.
type NotWritten struct {
	Path    string
	Problem format.Problem
}

// Renamed says what a rename reached and what it did not.
type Renamed struct {
	// Stencil is the fingerprint the stencil now stands at, which is what the
	// caller presents at its next write.
	Stencil domain.FileRef
	// Decks is every deck a heading was rewritten in, and Cards is how many
	// headings that was.
	Decks []string
	Cards int
	// NotWritten is one entry per deck the rename could not be written to.
	NotWritten []NotWritten
}

// RenameField gives one of a stencil's fields a different name.
//
// A field's name is written where the stencil declares it, in the braces of
// every face that places it, and as a heading in every card of every deck that
// stencil cuts. So the rename reaches them all. The stencil is written once,
// both of its names in the one write; then every deck of the vault is read, the
// heading is rewritten in the cards of that stencil, and what stands under it is
// untouched.
//
// Every field a stencil declares is written this way, the first included.
type RenameField struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Notes   port.NoteQueries
	// Links answers where the wikilink a card names its stencil by lands, which
	// is what says the card is cut by this stencil. A build holding none reaches
	// no card.
	Links port.LinkQueries
	Index func(ctx context.Context, v domain.Vault, paths []string) error
	// Now is when this is happening. An identifier written here carries it.
	Now func() time.Time
}

// stamp is the identifier a file this rename writes is to carry where it
// carries none.
func (u RenameField) stamp() (string, error) {
	at := time.Now
	if u.Now != nil {
		at = u.Now
	}
	return ulid.New(at())
}

// Execute renames the field, in the stencil first and then in the vault.
//
// The stencil leads: a rename the stencil refused reaches no deck. A deck it
// could not be written to keeps the old heading and comes back as a problem
// against that deck, and the decks after it are written all the same.
func (u RenameField) Execute(ctx context.Context, v domain.Vault, in Field) (Renamed, error) {
	if in.From == in.To {
		return Renamed{Stencil: in.At}, nil
	}

	out, err := u.rename(ctx, v, in)
	if err != nil || u.Index == nil {
		return out, err
	}
	return out, u.Index(ctx, v, append([]string{in.Stencil}, out.Decks...))
}

// rename is the whole of the writing, under this vault's write lock from before
// the stencil is read until after the last deck is replaced.
func (u RenameField) rename(ctx context.Context, v domain.Vault, in Field) (Renamed, error) {
	release, err := u.Writers.Hold(ctx, v)
	if err != nil {
		return Renamed{}, err
	}
	defer release()

	reader, err := u.Readers.Open(v)
	if err != nil {
		return Renamed{}, err
	}
	writer, err := u.Writers.Open(v)
	if err != nil {
		return Renamed{}, err
	}

	var out Renamed
	out.Stencil, err = u.stencil(ctx, reader, writer, in)
	if err != nil {
		return Renamed{}, err
	}

	paths, err := u.decks(ctx, v)
	if err != nil {
		return out, err
	}
	// The name the cards write is the name the stencil is linked by.
	named := domain.Basename(in.Stencil)
	one := deck{
		reader: reader, writer: writer, links: u.Links, vault: v,
		read: Read{Readers: u.Readers, Links: u.Links}, stamp: u.stamped,
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		cards, err := one.rename(ctx, path, in)
		if err != nil {
			out.NotWritten = append(out.NotWritten, NotWritten{Path: path, Problem: format.OnFile(
				format.CheckNotWritten,
				"a field renamed in "+named+" did not reach this deck: "+err.Error(),
			)})
			continue
		}
		if cards > 0 {
			out.Decks = append(out.Decks, path)
			out.Cards += cards
		}
	}
	return out, nil
}

// stencil writes the new name where the stencil declares the field.
func (u RenameField) stencil(
	ctx context.Context, reader port.VaultReader, writer port.VaultWriter, in Field,
) (domain.FileRef, error) {
	against := in.At
	if against == (domain.FileRef{}) {
		on, err := reader.Stat(ctx, in.Stencil)
		if err != nil {
			return domain.FileRef{}, fmt.Errorf("look at %s: %w", in.Stencil, err)
		}
		against = on
	}

	raw, err := reader.Read(ctx, in.Stencil)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("read %s: %w", in.Stencil, err)
	}
	f, err := format.OpenStencil(raw)
	if err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", in.Stencil, err)
	}
	if err := f.RenameField(in.From, in.To); err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", in.Stencil, err)
	}
	if err := u.stamped(f.Stamped); err != nil {
		return domain.FileRef{}, fmt.Errorf("%s: %w", in.Stencil, err)
	}
	return writer.Write(ctx, in.Stencil, f.Bytes(), against)
}

// stamped writes an identifier into a file that carries none, which is what
// the application changing what a note holds does.
func (u RenameField) stamped(into func(string) (bool, error)) error {
	identifier, err := u.stamp()
	if err != nil {
		return err
	}
	_, err = into(identifier)
	return err
}

// decks is every deck the vault holds, by path. The index answers which notes
// those are, in one question and without a file being opened.
//
// This runs under the vault's write lock, so a question per note in the vault
// is every other write in the application waiting behind it.
func (u RenameField) decks(ctx context.Context, v domain.Vault) ([]string, error) {
	return u.Notes.OfType(ctx, v.ID, domain.TypeDeck)
}

// deck is one deck's read and write, so that what could not be done to it is
// one error and the decks after it are written all the same.
type deck struct {
	reader port.VaultReader
	writer port.VaultWriter
	links  port.LinkQueries
	vault  domain.Vault
	// read is what opens the stencils this deck's cards are cut by, which is
	// what says which of a card's fields is first.
	read  Read
	stamp func(func(string) (bool, error)) error
}

// rename rewrites the heading in every card of this deck the stencil cuts, and
// reports how many it rewrote. A deck holding none of them is not written.
//
// A card is cut by the stencil its own wikilink lands on, which is what reading
// the deck against its stencils holds to.
//
// This is the application writing the file, so the deck it leaves behind is
// whole: it is stamped with an identifier where it carried none, and every card
// of it is given its mark and its heading.
func (d deck) rename(ctx context.Context, path string, in Field) (int, error) {
	on, err := d.reader.Stat(ctx, path)
	if err != nil {
		return 0, err
	}
	if on.Size > MaxBytes {
		return 0, fmt.Errorf("it is %d bytes, and %d is the most a deck is read at", on.Size, MaxBytes)
	}

	raw, err := d.reader.Read(ctx, path)
	if err != nil {
		return 0, err
	}
	f, err := format.OpenDeck(raw)
	if err != nil {
		return 0, err
	}
	at, err := cutting(ctx, d.links, d.vault.ID, path, f.Deck(on))
	if err != nil {
		return 0, err
	}
	cards, err := f.RenameField(at, in.Stencil, in.From, in.To)
	if err != nil {
		return 0, err
	}
	if cards == 0 {
		return 0, nil
	}
	by, _, err := d.read.stencils(ctx, d.vault, at)
	if err != nil {
		return 0, err
	}
	if _, err := f.Whole(by, mark.New); err != nil {
		return 0, err
	}
	if err := d.stamp(f.Stamped); err != nil {
		return 0, err
	}
	if _, err := d.writer.Write(ctx, path, f.Bytes(), on); err != nil {
		return 0, err
	}
	return cards, nil
}
