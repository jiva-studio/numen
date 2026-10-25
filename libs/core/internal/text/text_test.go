package text_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// getOffset is where a passage begins in a document's text.
func getOffset(t *testing.T, doc *text.Document, passage string) int {
	t.Helper()
	offset := strings.Index(doc.Text, passage)
	if offset < 0 {
		t.Fatalf("the document does not say %q", passage)
	}
	return offset
}

func checkLocation(t *testing.T, doc *text.Document, passage, want string) {
	t.Helper()
	if got := doc.Locate(getOffset(t, doc, passage)); got != want {
		t.Errorf("%q is located at %q, want %q", passage, got, want)
	}
}

func TestReadComposedReturnsArticleBytes(t *testing.T) {
	const article = "Tidal mills turned with the sea.\n\nThe pond filled at high water."
	store := artifactMap{
		domain.NewArtifactName(domain.ArtifactText, domain.Article, "abc123"):        []byte(article),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.Article, "abc123"): []byte("not the article"),
	}

	doc, err := text.ReadComposed(t.Context(), store, domain.Article, "abc123", []byte(article))

	if err != nil || doc.Text != article {
		t.Fatalf("read %q (%v), want the article as it was written", doc.Text, err)
	}
}
