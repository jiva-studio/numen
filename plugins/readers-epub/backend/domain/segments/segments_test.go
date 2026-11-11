package segments

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub"
)

func TestLongTextIsSplitIntoPartsOnWholeCharacters(t *testing.T) {
	text := strings.Repeat("ж", PartBytes) + "end"
	sections := []epub.Part{{Title: "One", Offset: 0}, {Title: "Inner", Offset: 0}, {Title: "Two", Offset: 2 * PartBytes}}

	parts, err := Cut(text, sections, epub.Places{})
	if err != nil {
		t.Fatal(err)
	}

	var joined strings.Builder
	var named []string
	for _, part := range parts {
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
