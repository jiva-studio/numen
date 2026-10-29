package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

const pluginID = "md.numen.readers.epub"

// newTextVault is a vault holding one file under name, the texts of that vault with the plugin
// providing, and the plugin's text server.
func newTextVault(t *testing.T, name string, raw []byte) (*hosttest.Texts, *TextServer) {
	t.Helper()
	files := hosttest.New("vault")
	files.File(name, string(raw))
	texts := hosttest.NewTexts("vault", pluginID)
	texts.File(name, raw)
	server := NewTextServer(files, texts, texts.GetStore(pluginID), hosttest.NewQueue())
	t.Cleanup(server.Close)
	return texts, server
}

// createText starts the text of a book and waits for it to be written.
func createText(t *testing.T, server *TextServer, req host.TextCreation) {
	t.Helper()
	if err := server.CreateText(t.Context(), req); err != nil {
		t.Fatalf("CreateText: %v", err)
	}
	server.Wait()
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
	raw := createImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)
	book, err := epub.Read(raw)
	if err != nil {
		t.Fatal(err)
	}

	createText(t, server, host.TextCreation{Path: "book.epub"})

	text := readLive(t, texts, "book.epub")
	if text.Version().Kind != textKind || !text.Version().IsFromFile() {
		t.Errorf("version %+v, want a book text made from the file", text.Version())
	}
	if text.Version().SourceHash != domain.HashContent(raw) || text.Version().File != domain.HashContent(raw)+".jsonl" {
		t.Errorf("version %+v does not name the file and the text as they are", text.Version())
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

			createText(t, server, host.TextCreation{Path: name})

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

func TestLongTextIsSplitIntoPartsOnWholeCharacters(t *testing.T) {
	text := strings.Repeat("ж", partBytes) + "end"
	sections := []epub.Part{{Title: "One", Offset: 0}, {Title: "Inner", Offset: 0}, {Title: "Two", Offset: 2 * partBytes}}

	parts := getSegments(text, sections, epub.Places{})

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

func TestTheTextOfABookIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	raw := createImageEpub(t)
	physics := hosttest.New("Physics")
	chemistry := hosttest.New("Chemistry")
	physics.Beside(chemistry)
	chemistry.File("book.epub", string(raw))
	texts := hosttest.NewTexts("Physics", pluginID)
	other := texts.With("Chemistry").(*hosttest.Texts)
	other.File("book.epub", raw)
	other.SetStore(pluginID, chemistry.Store())
	server := NewTextServer(physics, texts, physics.Store(), hosttest.NewQueue())
	t.Cleanup(server.Close)

	createText(t, server, host.TextCreation{Vault: "Chemistry", Path: "book.epub"})

	if text := readLive(t, other, "book.epub"); text.Version().Kind != textKind {
		t.Errorf("Chemistry's live version is %+v", text.Version())
	}
	if inShown, _ := texts.ListVersions(t.Context(), "book.epub"); len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
}

func TestEachSegmentIsSelectedByTheCFIOfItsFirstCharacter(t *testing.T) {
	book, err := epub.Read(createGuidedEpub(t))
	if err != nil {
		t.Fatal(err)
	}

	segments := getSegments(book.Text, book.Parts, book.ReadPlaces())

	var got []string
	for _, segment := range segments {
		selector, err := host.ParseSelector(segment.Selector)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, segment.Heading+"|"+firstWord(segment.Text)+"|"+selector.CFI)
	}
	want := []string{
		"|Prologue|epubcfi(/8/2!/4/2/1:0)",
		"|word|epubcfi(/8/2!/4/4/1:3)",
		"|ord|epubcfi(/8/2!/4/4/1:65539)",
		"|Epilogue|epubcfi(/8/4!/4/2/1:0)",
	}
	if strings.Join(got, ";") != strings.Join(want, ";") {
		t.Errorf("segments %q, want %q", got, want)
	}
}

// firstWord is the first word of a text, and empty for one with none.
func firstWord(text string) string {
	if words := strings.Fields(text); len(words) > 0 {
		return words[0]
	}
	return ""
}

func TestTheTextOfABookCarriesTheCFIOfItsCharacters(t *testing.T) {
	raw := createImageEpub(t)
	texts, server := newTextVault(t, "book.epub", raw)

	createText(t, server, host.TextCreation{Path: "book.epub"})

	text := readLive(t, texts, "book.epub")
	selector, err := host.ParseSelector(text.GetSelector(strings.Index(text.Text, "Goodbye")))
	if err != nil || selector.CFI != "epubcfi(/6/4!/4/4/1:0)" {
		t.Errorf("the selector of Goodbye is %+v (%v), want epubcfi(/6/4!/4/4/1:0)", selector, err)
	}
}
