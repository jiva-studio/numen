package epub_test

import (
	"os"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
)

// What opening a book costs, measured on the largest book of the corpus. A
// reader draws one document, so the document is measured apart from the book.
//
// The corpus is not in the repository, and these stand aside without it.

// largest is the corpus book with the most bytes on disk.
func largest(b *testing.B) string {
	b.Helper()
	root := os.Getenv(corpusEnv)
	if root == "" {
		root = relativeCorpus
	}
	if _, err := os.Stat(root); err != nil {
		b.Skipf("the corpus is not here: %v (set %s to say where it is)", err, corpusEnv)
	}

	held, err := booksUnder(root)
	if err != nil {
		b.Fatalf("walk the corpus: %v", err)
	}
	at, most := "", int64(0)
	for _, held := range held {
		info, err := os.Stat(held)
		if err != nil {
			continue
		}
		if info.Size() > most {
			at, most = held, info.Size()
		}
	}
	if at == "" {
		b.Skipf("the corpus at %s holds no books", root)
	}
	return at
}

func BenchmarkReadingABook(b *testing.B) {
	raw, err := os.ReadFile(largest(b))
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
	raw, err := os.ReadFile(largest(b))
	if err != nil {
		b.Fatal(err)
	}
	book, err := epub.Read(raw)
	if err != nil {
		b.Fatal(err)
	}
	// The longest document of the book, which is what a reader waits on.
	at, most := "", 0
	for _, doc := range book.Documents {
		if doc.Length > most {
			at, most = doc.Path, doc.Length
		}
	}
	b.ResetTimer()
	for b.Loop() {
		drawn, err := book.Markup(at)
		if err != nil {
			b.Fatal(err)
		}
		_ = drawn.HTML()
	}
}

func BenchmarkCountingThePages(b *testing.B) {
	raw, err := os.ReadFile(largest(b))
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
