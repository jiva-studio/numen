package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

const pluginID = "md.numen.readers.epub"

// newTextVault is a vault holding one file under name, the texts of that vault with the plugin
// providing, and the plugin's text server.
func newTextVault(t *testing.T, name string, raw []byte) (*hosttest.Texts, *TextServer) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, name), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	server := NewTextServer(func() string { return root }, texts, texts.GetStore(pluginID))
	return texts, server
}

// readLive is the live version of a file and its whole text.
func readLive(t *testing.T, texts *hosttest.Texts, path string) domain.VersionText {
	t.Helper()
	listing, err := texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range listing.Versions {
		if v.GetID() == listing.Live {
			text, err := host.ReadWholeText(t.Context(), texts, v)
			if err != nil {
				t.Fatal(err)
			}
			return text
		}
	}
	t.Fatalf("%s has no live version among %v", path, listing.Versions)
	return domain.VersionText{}
}

func TestTheTextOfABookIsItsLiveVersion(t *testing.T) {
	raw := createPictureEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)
	book, err := epub.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	if err := server.CreateText(t.Context(), host.TextCreation{Path: "book.epub"}); err != nil {
		t.Fatalf("CreateText: %v", err)
	}

	text := readLive(t, texts, "book.epub")
	if text.Version.Kind != textKind || !text.Version.IsFromFile() {
		t.Errorf("version %+v, want a book text made from the file", text.Version)
	}
	if text.Version.SourceHash != domain.HashContent(raw) || text.Version.File != domain.HashContent(raw)+".jsonl" {
		t.Errorf("version %+v does not name the file and the text as they are", text.Version)
	}
	if text.Text != book.Text {
		t.Errorf("text %q, want the text the reader sets the book with, %q", text.Text, book.Text)
	}
	sections := text.GetHeadings()
	if len(sections) != len(book.Parts) {
		t.Fatalf("sections %v, want the %d parts the book names", sections, len(book.Parts))
	}
	for i, part := range book.Parts {
		if sections[i].Title != part.Title || sections[i].Offset != part.Offset {
			t.Errorf("section %d is %q at %d, want %q at %d", i, sections[i].Title, sections[i].Offset, part.Title, part.Offset)
		}
	}
	if got := text.Locate(strings.Index(text.Text, "Goodbye")); got != "Second part" {
		t.Errorf("Goodbye is located at %q, want Second part", got)
	}
}

func TestAFileWithNoReadableBookGetsNoVersion(t *testing.T) {
	for name, raw := range map[string][]byte{
		"corrupted.epub": []byte("INVALID_EPUB_ZIP"),
		"locked.epub":    createEncryptedEpub(t),
	} {
		t.Run(name, func(t *testing.T) {
			texts, server := newTextVault(t, name, raw)

			if err := server.CreateText(t.Context(), host.TextCreation{Path: name}); err != nil {
				t.Fatalf("CreateText: %v", err)
			}

			listing, err := texts.ListVersions(t.Context(), name)
			if err != nil {
				t.Fatal(err)
			}
			if len(listing.Versions) != 0 {
				t.Errorf("versions %v, want none", listing.Versions)
			}
		})
	}
}

func TestTheBookReaderMakesTextOfFilesOnly(t *testing.T) {
	_, server := newTextVault(t, "book.epub", createPictureEpub(t))

	err := server.CreateText(t.Context(), host.TextCreation{Path: "book.epub", Source: "md.numen.tools.proofreading/fix"})

	if !errors.Is(err, host.ErrInvalidVersion) {
		t.Errorf("CreateText of a version = %v, want ErrInvalidVersion", err)
	}
}

func TestLongTextIsSplitIntoPartsOnWholeCharacters(t *testing.T) {
	text := strings.Repeat("ж", partBytes) + "end"
	sections := []epub.Part{{Title: "One", Offset: 0}, {Title: "Inner", Offset: 0}, {Title: "Two", Offset: 2 * partBytes}}

	parts := getBookParts(text, sections)

	var joined strings.Builder
	var named []string
	for _, part := range parts {
		if len(part.Text) > partBytes {
			t.Errorf("a part of %d bytes, want at most %d", len(part.Text), partBytes)
		}
		if part.Text != "" && !utf8.RuneStart(part.Text[0]) {
			t.Errorf("a part begins inside a character")
		}
		if part.Heading != "" {
			named = append(named, part.Heading)
		}
		joined.WriteString(part.Text)
	}
	if joined.String() != text {
		t.Error("the parts do not join into the text")
	}
	if strings.Join(named, ",") != "One,Inner,Two" {
		t.Errorf("sections %v, want One, Inner and Two", named)
	}
	version := domain.NewVersionText(domain.TextVersion{Kind: textKind}, parts)
	if got := version.GetHeadingsAt(0); strings.Join(got, ",") != "One,Inner" {
		t.Errorf("names at 0 are %v, want One and Inner", got)
	}
	if got := version.GetHeadings(); got[2].Offset != 2*partBytes {
		t.Errorf("Two begins at %d, want %d", got[2].Offset, 2*partBytes)
	}
}
