package epub_test

import (
	"os"
	"testing"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// What opening a book costs, measured on the largest book of the corpus. A
// reader draws one document, so the document is measured apart from the book.
//
// The corpus is not in the repository, and these stand aside without it.

// getLargestBook is the corpus book with the most bytes on disk.
func getLargestBook(b *testing.B) string {
	b.Helper()
	root := os.Getenv(corpusEnv)
	if root == "" {
		root = relativeCorpus
	}
	if _, err := os.Stat(root); err != nil {
		b.Skipf("the corpus is not here: %v (set %s to say where it is)", err, corpusEnv)
	}

	target, err := booksUnder(root)
	if err != nil {
		b.Fatalf("walk the corpus: %v", err)
	}
	pos, most := "", int64(0)
	for _, targetVal := range target {
		info, err := os.Stat(targetVal)
		if err != nil {
			continue
		}
		if info.Size() > most {
			pos, most = targetVal, info.Size()
		}
	}
	if pos == "" {
		b.Skipf("the corpus at %s holds no books", root)
	}
	return pos
}

func BenchmarkReadingABook(b *testing.B) {
	raw, err := os.ReadFile(getLargestBook(b))
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for b.Loop() {
		if _, err := epub.Read(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDrawingOneDocument(b *testing.B) {
	raw, err := os.ReadFile(getLargestBook(b))
	if err != nil {
		b.Fatal(err)
	}
	book, err := epub.Read(raw)
	if err != nil {
		b.Fatal(err)
	}
	// The longest document of the book, which is what a reader waits on.
	pos, most := "", 0
	for _, doc := range book.Documents {
		if doc.Length > most {
			pos, most = doc.Path, doc.Length
		}
	}
	b.ResetTimer()
	for b.Loop() {
		rendered, err := book.Markup(pos)
		if err != nil {
			b.Fatal(err)
		}
		_ = rendered.HTML()
	}
}

func BenchmarkCountingThePages(b *testing.B) {
	raw, err := os.ReadFile(getLargestBook(b))
	if err != nil {
		b.Fatal(err)
	}
	book, err := epub.Read(raw)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_ = book.PageCount()
	}
}
