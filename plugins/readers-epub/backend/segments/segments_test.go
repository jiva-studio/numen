package segments

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/testsupport"
)

func TestLongTextIsSplitIntoPartsOnWholeCharacters(t *testing.T) {
	if PartBytes != 65536 {
		t.Fatalf("PartBytes = %d, want 65536", PartBytes)
	}
	text := strings.Repeat("ж", PartBytes) + "end"
	sections := []epub.Part{{Title: "One", Offset: 0}, {Title: "Inner", Offset: 0}, {Title: "Two", Offset: 2 * PartBytes}}

	parts, err := Cut(text, sections, epub.Places{})
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	var named []string
	for _, part := range parts {
		if part.Selector != "" {
			t.Errorf("empty places produced selector %q, want empty", part.Selector)
		}
		if len(part.Text) > PartBytes {
			t.Errorf("a part of %d bytes, want at most %d", len(part.Text), PartBytes)
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
	version := domain.NewVersionText(domain.TextVersion{Kind: "book-text"}, parts)
	if got := version.GetHeadingsAt(0); strings.Join(got, ",") != "One,Inner" {
		t.Errorf("names at 0 are %v, want One and Inner", got)
	}
	if got := version.GetHeadings(); got[2].Offset != 2*PartBytes {
		t.Errorf("Two begins at %d, want %d", got[2].Offset, 2*PartBytes)
	}
}

func TestSegmentBoundaries(t *testing.T) {
	parts, err := Cut(
		"FirstSecondThird",
		[]epub.Part{{Title: "Intro", Offset: 0}, {Title: "End", Offset: 11}},
		epub.Places{Blocks: []int{0, 5, 5, 16}},
	)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range parts {
		got = append(got, p.Heading+":"+p.Text)
	}
	want := []string{"Intro:", ":First", ":Second", "End:", ":Third"}
	if !slices.Equal(got, want) {
		t.Errorf("Cut parts = %v, want %v", got, want)
	}

	longParts, err := Cut(strings.Repeat("a", PartBytes+20), nil, epub.Places{Blocks: []int{PartBytes + 10}})
	if err != nil {
		t.Fatal(err)
	}
	if len(longParts) != 3 || len(longParts[0].Text) != PartBytes || len(longParts[1].Text) != 10 || len(longParts[2].Text) != 10 {
		t.Errorf("longParts lengths = %d, %d, %d (total %d parts), want %d, 10, 10",
			len(longParts[0].Text), len(longParts[1].Text), len(longParts[2].Text), len(longParts), PartBytes)
	}

	book, err := epub.Read(testsupport.CreateImageEpub(t))
	if err != nil {
		t.Fatal(err)
	}
	placedParts, err := Cut(book.Text, book.Parts, book.ReadPlaces())
	if err != nil {
		t.Fatal(err)
	}
	if len(placedParts) == 0 || !strings.Contains(string(placedParts[0].Selector), "epubcfi(/6/2!/4/2/1:0)") {
		t.Errorf("first placed part selector = %q, want epubcfi(/6/2!/4/2/1:0)", placedParts[0].Selector)
	}
}
