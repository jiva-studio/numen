package testsupport

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// TextReader is the plugin every text version Texts gives was made by.
const TextReader = "test.reader"

// documentText is the text of every document Texts reads: two pages.
const documentText = "Alpha Beta\nGamma Delta\n"

// Texts is the text versions of the vault's documents and books: each has one, made by TextReader
// from the bytes it holds now. A file that opens with the PDF marker is positioned text, a page a
// part; one NewBook made is plain text, a line a section.
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
	version := domain.TextVersion{Path: path, PluginID: TextReader, Key: hash[:12], SourceHash: hash, Hash: hash}
	if bytes.HasPrefix(raw, []byte("%PDF-")) {
		version.Kind = "page-text"
		return domain.NewVersionText(version, splitAt(documentText, []int{0, 11}, nil)), nil
	}
	if !bytes.HasPrefix(raw, []byte(bookMark)) {
		return domain.VersionText{}, textport.ErrNoText
	}
	text := string(raw[len(bookMark):])
	var starts []int
	var titles []string
	offset := 0
	for i, line := range strings.Split(text, "\n") {
		starts = append(starts, offset)
		titles = append(titles, "Part "+strconv.Itoa(i+1))
		offset += len(line) + 1
	}
	version.Kind = "book-text"
	return domain.NewVersionText(version, splitAt(text, starts, titles)), nil
}

// splitAt cuts text into segments at the given starts. A segment with a title
// opens a heading; one without is a page, numbered by position from 1.
func splitAt(text string, starts []int, titles []string) []domain.TextSegment {
	parts := make([]domain.TextSegment, 0, len(starts))
	for i, from := range starts {
		to := len(text)
		if i+1 < len(starts) {
			to = starts[i+1]
		}
		part := domain.TextSegment{Text: text[from:to], Selector: domain.Selector(`{"page":` + strconv.Itoa(i+1) + `}`)}
		if i < len(titles) {
			part.Heading = titles[i]
		}
		parts = append(parts, part)
	}
	return parts
}

func (t vaultTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	doc, err := t.GetLiveText(ctx, path, fileHash)
	return doc.Version().GetID(), err
}

func (t vaultTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version().SourceHash != fileHash {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (t vaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	doc, err := t.read(ctx, path)
	if err == nil && doc.Version().GetID() != id {
		return domain.VersionText{}, textport.ErrNoText
	}
	return doc, err
}

func (vaultTexts) RequestText(context.Context, string, string) error { return nil }

// IsTextSource reports whether a file has text, where Texts reads a version of it.
func (t vaultTexts) IsTextSource(ctx context.Context, path string) (bool, error) {
	_, err := t.read(ctx, path)
	return err == nil, nil
}
