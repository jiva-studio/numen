package vectors

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// bookTexts gives every book the stand-in reads one text version, made from the
// bytes the vault holds now.
type bookTexts struct{ readers vaultport.VaultReaders }

func (b bookTexts) Open(v domain.Vault) (textport.VaultTexts, error) {
	files, err := b.readers.Open(v)
	if err != nil {
		return nil, err
	}
	return vaultBookTexts{files: files}, nil
}

type vaultBookTexts struct{ files vaultport.VaultReader }

// read is the version of the book at a path, made from the bytes it holds now.
func (t vaultBookTexts) read(ctx context.Context, path string) (domain.VersionText, error) {
	raw, err := t.files.Read(ctx, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	book, err := documents{}.ReadBook(ctx, raw)
	if err != nil {
		return domain.VersionText{}, textport.ErrNoText
	}
	hash := domain.HashContent(raw)
	version := domain.TextVersion{Path: path, PluginID: "test.books", Key: hash[:8], Kind: "book-text", SourceHash: hash}
	var parts []domain.TextSegment
	for i, part := range book.Parts {
		end := len(book.Text)
		if i+1 < len(book.Parts) {
			end = book.Parts[i+1].Offset
		}
		parts = append(parts, domain.TextSegment{Text: book.Text[part.Offset:end], Heading: part.Title})
	}
	return domain.NewVersionText(version, parts), nil
}

func (t vaultBookTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	doc, err := t.GetLiveText(ctx, path, fileHash)
	return doc.Version.GetID(), err
}

func (t vaultBookTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version.SourceHash != fileHash {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (t vaultBookTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version.GetID() != id {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (vaultBookTexts) RequestText(context.Context, string, string) error { return nil }
