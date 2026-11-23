package books

import (
	"bytes"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/internal/testsupport"
)

func newCountingReader(t *testing.T, files *hosttest.Vault) (*Reader, *atomic.Int64) {
	t.Helper()
	books := NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	t.Cleanup(books.Close)
	var opens atomic.Int64
	books.books.readFile = func(ctx context.Context, v host.FileReader, path string) ([]byte, domain.FileStat, error) {
		opens.Add(1)
		return v.Read(ctx, path)
	}
	return books, &opens
}

func readEntryFrom(books *Reader, name, entry string) ([]byte, error) {
	return books.ReadBookEntry(context.Background(),
		name, entry, nil)
}

func TestReadBookEntryOpensTheFileOnceForManyEntries(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, opens := newCountingReader(t, files)
	for range 50 {
		for _, entry := range []string{"EPUB/cover.png", "EPUB/c1.xhtml", "EPUB/c2.xhtml"} {
			if _, err := readEntryFrom(books, name, entry); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := opens.Load(); got != 1 {
		t.Errorf("the file was opened %d times for 150 entries, want 1", got)
	}
}

func TestReadBookEntryReopensTheBookWhenTheFileChanged(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, opens := newCountingReader(t, files)
	if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	changed := testsupport.CreateZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Changed</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest></manifest>
  <spine></spine>
</package>`),
		"EPUB/cover.png": []byte("\x89PNG\r\n\x1a\nanother picture, longer than the first one was"),
	})
	files.WriteFile(name, string(changed))
	body, err := readEntryFrom(books, name, "EPUB/cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(body, testsupport.GetPNGBytes()) {
		t.Error("the entry is the picture of the book before the file changed")
	}
	if got := opens.Load(); got != 2 {
		t.Errorf("the file was opened %d times, want 2", got)
	}
}

func TestReadBookEntryEvictsTheLeastRecentlyReadBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	books, opens := newCountingReader(t, files)
	for i := range maxCachedBooks {
		name := "copy" + string(rune('a'+i)) + ".epub"
		files.WriteFile(name, string(testsupport.CreateImageEpub(t)))
		if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
			t.Fatal(err)
		}
	}
	first := books.books.books[0]
	second := books.books.books[1]
	extra := "copy" + string(rune('a'+maxCachedBooks)) + ".epub"
	files.WriteFile(extra, string(testsupport.CreateImageEpub(t)))
	if _, err := readEntryFrom(books, extra, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if !first.evicted || first.archive != nil {
		t.Errorf("least recently read book evicted=%v archive=%v, want evicted with nil archive", first.evicted, first.archive)
	}
	if second.evicted || second.archive == nil {
		t.Errorf("second book evicted=%v archive=%v, want kept in cache", second.evicted, second.archive)
	}
	if got := len(books.books.books); got != maxCachedBooks {
		t.Fatalf("%d books are open, want %d", got, maxCachedBooks)
	}
	// The first book was the least recently read and was evicted.
	if _, err := readEntryFrom(books, "copya.epub", "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if got := opens.Load(); got != maxCachedBooks+2 {
		t.Errorf("the files were opened %d times, want %d", got, maxCachedBooks+2)
	}
}

func TestReadBookEntryAnswersConcurrentRequests(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, opens := newCountingReader(t, files)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 20 {
				body, err := readEntryFrom(books, name, "EPUB/cover.png")
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(body, testsupport.GetPNGBytes()) {
					t.Errorf("the entry is %q, want the picture's bytes", body)
					return
				}
			}
		})
	}
	wg.Wait()
	if got := opens.Load(); got != 1 {
		t.Errorf("the file was opened %d times, want 1", got)
	}
}

func TestCloseClosesTheFilesOfTheCachedBooks(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, _ := newCountingReader(t, files)
	if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	book := books.books.books[0]
	books.Close()
	if book.archive != nil {
		t.Error("the archive of the book is still kept after Close")
	}
	_, err := readEntryFrom(books, name, "EPUB/cover.png")
	if codeOf(err) != KindUnavailable {
		t.Fatalf("code = %v (%v), want unavailable", codeOf(err), err)
	}
}

func TestCloseKeepsTheFileOpenForAReadingRequest(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, _ := newCountingReader(t, files)
	book, err := books.books.acquire(t.Context(), files, "", name)
	if err != nil {
		t.Fatal(err)
	}
	books.Close()
	body, err := book.archive.Entry("EPUB/cover.png")
	if err != nil || !bytes.Equal(body, testsupport.GetPNGBytes()) {
		t.Fatalf("entry = %q, %v, want the picture's bytes while the request reads", body, err)
	}
	books.books.release(book)
	if book.archive != nil {
		t.Error("the archive is still kept after the last request gave the book back")
	}
}

func TestCacheLRUAndConcurrentEviction(t *testing.T) {
	files, name, _, _ := writeBook(t)
	files.WriteFile("second.epub", string(testsupport.CreateImageEpub(t)))
	books, opens := newCountingReader(t, files)

	// Open name, then second.epub, then name again to exercise LRU loop skipping non-matching key
	if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := readEntryFrom(books, "second.epub", "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := readEntryFrom(books, "second.epub", "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if got := opens.Load(); got != 2 {
		t.Fatalf("expected 2 opens, got %d", got)
	}
	if got := len(books.books.books); got != 2 {
		t.Fatalf("expected 2 cached books after cache hit, got %d", got)
	}
	if _, err := readEntryFrom(books, "missing.epub", "EPUB/cover.png"); codeOf(err) != KindNotFound || opens.Load() != 2 {
		t.Fatalf("missing book err = %v, opens = %d, want CodeNotFound without calling readFile", err, opens.Load())
	}

	// Acquire second.epub and hold it while changing its size only (or mtime only)
	held, err := books.books.acquire(t.Context(), files, "", "second.epub")
	if err != nil {
		t.Fatal(err)
	}
	// Acquire again so readers == 2, then release one so readers == 1 (must not nil archive)
	held2, err := books.books.acquire(t.Context(), files, "", "second.epub")
	if err != nil {
		t.Fatal(err)
	}
	// Modify file size to force eviction while held
	files.WriteFile("second.epub", string(testsupport.CreateGuidedEpub(t)))
	if _, err := readEntryFrom(books, "second.epub", "EPUB/c1.xhtml"); err != nil {
		t.Fatal(err)
	}
	if !held.evicted || held.archive == nil {
		t.Fatalf("expected held book to be marked evicted with non-nil archive while readers=%d", held.readers)
	}
	books.books.release(held2)
	if held.archive == nil {
		t.Fatal("archive was cleared while one reader remained")
	}
	books.books.release(held)
	if held.archive != nil {
		t.Fatal("archive was not cleared after last reader released")
	}

	// Also test when size is identical but mtime differs
	b3, err := books.books.acquire(t.Context(), files, "", name)
	if err != nil {
		t.Fatal(err)
	}
	b3.mtime--
	books.books.release(b3)
	b4, err := books.books.acquire(t.Context(), files, "", name)
	if err != nil {
		t.Fatal(err)
	}
	books.books.release(b4)
	if !b3.evicted {
		t.Error("expected book with changed mtime to be evicted")
	}

	// Also test when mtime is identical but size differs
	b4.size--
	b5, err := books.books.acquire(t.Context(), files, "", name)
	if err != nil {
		t.Fatal(err)
	}
	books.books.release(b5)
	if !b4.evicted {
		t.Error("expected book with changed size to be evicted")
	}

	// Test openBook readFile failure after StatFile succeeds
	books.Close()
	books2 := NewReader(func(id domain.VaultID) host.FileReader { return files.With(id) })
	t.Cleanup(books2.Close)
	books2.books.readFile = func(context.Context, host.FileReader, string) ([]byte, domain.FileStat, error) {
		return nil, domain.FileStat{}, os.ErrNotExist
	}
	if _, err := readEntryFrom(books2, name, "EPUB/cover.png"); codeOf(err) != KindNotFound {
		t.Errorf("readFile error code = %v (%v), want KindNotFound", codeOf(err), err)
	}
}
