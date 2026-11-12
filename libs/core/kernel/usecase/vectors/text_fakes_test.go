package vectors

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// bookTexts gives every book the stand-in reads one text version, made from the
// bytes the vault holds now.
type bookTexts struct{ readers vaultport.VaultReaders }

func (b bookTexts) Open(v domain.Vault) (text.VaultTexts, error) {
	files, err := b.readers.Open(v)
	if err != nil {
		return nil, err
	}
	return vaultBookTexts{files: files}, nil
}

type vaultBookTexts struct{ files vaultport.VaultReader }

func (t vaultBookTexts) read(ctx context.Context, path string) (domain.VersionText, error) {
	raw, err := t.files.Read(ctx, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	parts, err := indexfake.ReadBookSegments(raw)
	if err != nil {
		return domain.VersionText{}, text.ErrNoText
	}
	hash := domain.HashContent(raw)
	version := domain.TextVersion{Path: path, PluginID: "test.books", Key: hash[:8], Kind: "book-text", InputHash: hash}
	return domain.NewVersionText(version, parts), nil
}

func (t vaultBookTexts) FindLiveVersion(ctx context.Context, path, contentHash string) (domain.TextVersionID, error) {
	doc, err := t.GetLiveText(ctx, path, contentHash)
	return doc.Version().GetID(), err
}

func (t vaultBookTexts) GetLiveText(ctx context.Context, path, contentHash string) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version().InputHash != contentHash {
		return domain.VersionText{}, text.ErrNoText
	}
	return doc, err
}

func (t vaultBookTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version().GetID() != id {
		return domain.VersionText{}, text.ErrNoText
	}
	return doc, err
}

func (vaultBookTexts) RequestText(context.Context, string, string) error { return nil }

func (t vaultBookTexts) IsTextSource(ctx context.Context, path string) (bool, error) {
	ref, err := t.files.StatFile(ctx, path)
	return err == nil && ref.Kind == domain.KindFile, nil
}

func (t vaultBookTexts) GetContentHash(ctx context.Context, path string) (_ string, err error) {
	file, err := t.files.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}
