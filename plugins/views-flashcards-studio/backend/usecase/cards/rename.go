package cards

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/ulid"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// Rename is which field of which card model is being renamed, and to what.
type Rename struct {
	CardModel string
	From      string
	To        string
	FileStat  domain.FileStat
}

// UnwrittenDeck is one deck a rename did not reach. It keeps the old heading.
type UnwrittenDeck struct {
	Path    string
	Problem format.Problem
}

// RenameResult reports what a rename changed and what it did not.
type RenameResult struct {
	CardModel  domain.FileStat
	Decks      []string
	Cards      int
	NotWritten []UnwrittenDeck
}

// RenameField renames one of a card model's fields, the first included.
// The name is written in the card model's declaration, in the braces of every template that
// places it, and as a heading in every card of every deck the card model generates. The card
// model is written once with both names; then every deck of the vault is read and the heading is
// rewritten in those cards. The content under a heading is not changed.
type RenameField struct {
	maxDecks int

	readers ports.VaultReaders
	writers ports.VaultWriters
	notes   TypeQueries
	links   ports.LinkQueries
	index   edit.IndexUpdater
	now     ports.Clock
	entropy Entropy
}

// TypeQueries is the single index query a rename needs. The rename runs entirely
// under the vault's write lock, so the queries it makes are limited: one query per
// note of the vault would block every other write behind it.
type TypeQueries interface {
	// 	// GetPathsOfType returns every note of one type in the vault, by path. A
	// 	// caller that needs the decks or the card models of a vault uses it, and no
	// 	// file is opened to determine the type of each note.
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
}

// NewRenameField renames a card model's field across its decks. A vault of more than maxDecks decks is
// refused; UnboundedDecks takes a vault of any size.
func NewRenameField(cfg Config, maxDecks int) RenameField {
	return RenameField{
		readers: cfg.Readers, writers: cfg.Writers, notes: cfg.Notes, links: cfg.Links,
		index: cfg.Index, now: cfg.Now, entropy: cfg.Entropy, maxDecks: maxDecks,
	}
}

func (u RenameField) createIdentifier() (string, error) {
	return ulid.NewGenerator(rand.Reader).NewID(u.now.Now())
}

// Execute renames the field, in the card model first and then in the vault.
//
// could not be written to keeps the old heading and is returned as a problem
// against that deck, and the decks after it are written all the same.
func (u RenameField) Execute(ctx context.Context, v domain.Vault, in Rename) (RenameResult, error) {
	if err := u.checkDeckCount(ctx, v); err != nil {
		return RenameResult{}, err
	}
	if in.From == in.To {
		return RenameResult{CardModel: in.FileStat, Decks: nil, Cards: 0, NotWritten: nil}, nil
	}

	out, err := u.rename(ctx, v, in)
	if err != nil || u.index == nil {
		return out, err
	}
	content := append([]string{in.CardModel}, out.Decks...)
	return out, edit.WrapIndexOutdated(u.index(ctx, v, content), content...)
}

// MaxDecksPerRename is the most decks an agent's rename reads and writes at once.
const MaxDecksPerRename = 200

// UnboundedDecks is the bound of a rename that takes a vault of any size.
const UnboundedDecks = math.MaxInt

func (u RenameField) checkDeckCount(ctx context.Context, v domain.Vault) error {
	if u.maxDecks <= 0 || u.maxDecks == UnboundedDecks {
		return nil
	}
	target, err := u.getDeckPaths(ctx, v)
	if err != nil {
		return err
	}
	if len(target) > u.maxDecks {
		return fmt.Errorf(
			"this rename reads and writes every deck in the vault, and there are %d of them, "+
				"which is more than the %d this does at once: rename the field in the window",
			len(target), u.maxDecks)
	}
	return nil
}

func (u RenameField) rename(ctx context.Context, v domain.Vault, in Rename) (result RenameResult, err error) {
	release, err := u.writers.Lock(ctx, v)
	if err != nil {
		return RenameResult{}, err
	}
	defer func() { err = errors.Join(err, release()) }()

	reader, err := u.readers.Open(v)
	if err != nil {
		return RenameResult{}, err
	}
	writer, err := u.writers.Open(v)
	if err != nil {
		return RenameResult{}, err
	}

	var out RenameResult
	out.CardModel, err = u.writeCardModelField(ctx, reader, writer, in)
	if err != nil {
		return RenameResult{}, err
	}

	paths, err := u.getDeckPaths(ctx, v)
	if err != nil {
		return out, err
	}
	// The name the cards write is the name the card model is linked by.
	named := domain.Basename(in.CardModel)
	one := deckWriter{
		reader: reader, writer: writer, links: u.links, vault: v,
		read: NewRead(u.readers, u.links), writeIdentifier: u.writeIdentifier, createCardID: newCardIDs(u.entropy),
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		cards, err := one.rename(ctx, path, in)
		if err != nil {
			out.NotWritten = append(out.NotWritten, UnwrittenDeck{Path: path, Problem: format.OnFile(
				format.FaultNotWritten,
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

func (u RenameField) writeCardModelField(
	ctx context.Context, reader ports.VaultReader, writer ports.VaultWriter, in Rename,
) (domain.FileStat, error) {
	expectedFp := in.FileStat
	if expectedFp.IsZero() {
		on, err := reader.StatFile(ctx, in.CardModel)
		if err != nil {
			return domain.FileStat{}, fmt.Errorf("look at %s: %w", in.CardModel, err)
		}
		expectedFp = on
	}

	raw, err := reader.Read(ctx, in.CardModel)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("read %s: %w", in.CardModel, err)
	}
	f, err := format.OpenCardModel(raw)
	if err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", in.CardModel, err)
	}
	if err := f.RenameField(in.From, in.To); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", in.CardModel, err)
	}
	// gomutants:disable-next-line BRANCH_IF reason="crypto/rand.Reader and freshly opened card model frontmatter do not fail when writing an identifier."
	if err := u.writeIdentifier(f.WriteIdentifier); err != nil {
		return domain.FileStat{}, fmt.Errorf("%s: %w", in.CardModel, err)
	}
	return writer.Write(ctx, in.CardModel, f.Bytes(), expectedFp)
}

func (u RenameField) writeIdentifier(into func(string) (bool, error)) error {
	identifier, err := u.createIdentifier()
	// gomutants:disable-next-line BRANCH_IF reason="crypto/rand.Reader does not fail in normal execution."
	if err != nil {
		return err
	}
	_, err = into(identifier)
	return err
}

func (u RenameField) getDeckPaths(ctx context.Context, v domain.Vault) ([]string, error) {
	return u.notes.GetPathsOfType(ctx, v.ID, format.TypeDeck)
}

// deckWriter is the read and write of one deck. A failure on one deck is reported
// as one error and the later decks are still written.
type deckWriter struct {
	reader          ports.VaultReader
	writer          ports.VaultWriter
	links           ports.LinkQueries
	vault           domain.Vault
	read            Read
	writeIdentifier func(func(string) (bool, error)) error
	createCardID    func() format.ID
}

func (d deckWriter) rename(ctx context.Context, path string, in Rename) (int, error) {
	on, err := d.reader.StatFile(ctx, path)
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	if on.Size > format.MaxDeckBytes {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, fmt.Errorf("it is %d bytes, and %d is the most a deck is read at", on.Size, format.MaxDeckBytes)
	}

	raw, err := d.reader.Read(ctx, path)
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	f, err := format.OpenDeck(raw)
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	pos, err := getCardModelPaths(ctx, d.links, d.vault.ID, path, f.Deck(on))
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	cards, err := f.RenameField(pos, in.CardModel, in.From, in.To)
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	if cards == 0 {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Caller only records the deck when cards > 0."
		return 0, nil
	}
	delta, _, err := d.read.readCardModels(ctx, d.vault, pos)
	if err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	// gomutants:disable-next-line BRANCH_IF reason="DeckFile.Whole does not fail on a deck that already passed OpenDeck and RenameField."
	if _, err := f.Whole(delta, d.createCardID); err != nil {
		return 0, err
	}
	if err := d.writeIdentifier(f.WriteIdentifier); err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	if _, err := d.writer.Write(ctx, path, f.Bytes(), on); err != nil {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Caller ignores card count when rename returns an error."
		return 0, err
	}
	return cards, nil
}
