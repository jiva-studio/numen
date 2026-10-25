package testsupport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// TextReader is the plugin every text version Texts gives was made by.
const TextReader = "test.reader"

// Texts is the text versions of the files Documents reads: each has one, made
// by TextReader from the bytes it holds now. A document is positioned text, a
// page a part; a book is plain text, a section a part.
type Texts struct {
	Readers vaultport.VaultReaders
}

func (t Texts) Open(v domain.Vault) (textport.VaultTexts, error) {
	files, err := t.Readers.Open(v)
	if err != nil {
		return nil, err
	}
	return vaultTexts{files: files}, nil
}

type vaultTexts struct {
	files vaultport.VaultReader
}

// read is the version of the file at a path, made from the bytes it holds now.
func (t vaultTexts) read(ctx context.Context, path string) (domain.VersionText, error) {
	raw, err := t.files.Read(ctx, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	hash := domain.HashContent(raw)
	version := domain.TextVersion{Path: path, PluginID: TextReader, Key: hash[:12], InputHash: hash, Hash: hash}
	if layer, err := (Documents{}).Read(ctx, raw); err == nil {
		version.Format = domain.TextFormatPositioned
		return domain.NewVersionText(version, splitAt(layer.Text, layer.Pages, nil)), nil
	}
	book, err := (Documents{}).ReadBook(ctx, raw)
	if err != nil {
		return domain.VersionText{}, textport.ErrNoText
	}
	version.Format = domain.TextFormatPlain
	starts := make([]int, 0, len(book.Parts))
	titles := make([]string, 0, len(book.Parts))
	for _, part := range book.Parts {
		starts = append(starts, part.Offset)
		titles = append(titles, part.Title)
	}
	return domain.NewVersionText(version, splitAt(book.Text, starts, titles)), nil
}

// splitAt is text cut into parts where each start says. A part with a title
// opens a section; a part without one is a page, numbered by its place.
func splitAt(text string, starts []int, titles []string) []domain.TextPart {
	parts := make([]domain.TextPart, 0, len(starts))
	for i, from := range starts {
		to := len(text)
		if i+1 < len(starts) {
			to = starts[i+1]
		}
		part := domain.TextPart{Text: text[from:to], Place: domain.TextPlace{Page: i}}
		if i < len(titles) {
			part.Section = titles[i]
		}
		parts = append(parts, part)
	}
	return parts
}

func (t vaultTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	doc, err := t.GetLiveText(ctx, path, fileHash)
	return doc.Version.GetID(), err
}

func (t vaultTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version.InputHash != fileHash {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (t vaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version.GetID() != id {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (vaultTexts) RequestText(context.Context, string, string) error { return nil }
