package service

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
)

// newCountingServer is a books on files that counts the files it reads.
func newCountingServer(t *testing.T, files *hosttest.Vault) (*BookServer, *atomic.Int64) {
	t.Helper()
	books := NewBookServer(files)
	t.Cleanup(books.Close)
	var opens atomic.Int64
	books.books.readFile = func(ctx context.Context, v host.Vault, path string) ([]byte, host.FileStat, error) {
		opens.Add(1)
		return v.Read(ctx, path)
	}
	return books, &opens
}

func readEntryFrom(books *BookServer, name, entry string) (api.ReadBookEntryResponse, error) {
	return books.ReadBookEntry(context.Background(),
		api.ReadBookEntryRequest{Path: name, Entry: entry})
}

func TestReadBookEntryOpensTheFileOnceForManyEntries(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, opens := newCountingServer(t, files)
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
	books, opens := newCountingServer(t, files)
	if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	changed := createZipEpub(t, map[string][]byte{
		"EPUB/package.opf": []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="id" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Changed</dc:title><dc:identifier id="id">x</dc:identifier></metadata>
  <manifest></manifest>
  <spine></spine>
</package>`),
		"EPUB/cover.png": []byte("\x89PNG\r\n\x1a\nanother picture, longer than the first one was"),
	})
	files.File(name, string(changed))
	resp, err := readEntryFrom(books, name, "EPUB/cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(resp.Body, pngBytes) {
		t.Error("the entry is the picture of the book before the file changed")
	}
	if got := opens.Load(); got != 2 {
		t.Errorf("the file was opened %d times, want 2", got)
	}
}

func TestReadBookEntryEvictsTheLeastRecentlyReadBook(t *testing.T) {
	files, _, _, _ := writeBook(t)
	books, opens := newCountingServer(t, files)
	for i := range maxCachedBooks + 1 {
		name := "copy" + string(rune('a'+i)) + ".epub"
		files.File(name, string(createImageEpub(t)))
		if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
			t.Fatal(err)
		}
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
	books, opens := newCountingServer(t, files)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 20 {
				resp, err := readEntryFrom(books, name, "EPUB/cover.png")
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(resp.Body, pngBytes) {
					t.Errorf("the entry is %q, want the picture's bytes", resp.Body)
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
	books, _ := newCountingServer(t, files)
	if _, err := readEntryFrom(books, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	book := books.books.books[0]
	books.Close()
	if book.archive != nil {
		t.Error("the archive of the book is still kept after Close")
	}
	_, err := readEntryFrom(books, name, "EPUB/cover.png")
	if codeOf(err) != server.CodeUnavailable {
		t.Fatalf("code = %v (%v), want unavailable", codeOf(err), err)
	}
}

func TestCloseKeepsTheFileOpenForAReadingRequest(t *testing.T) {
	files, name, _, _ := writeBook(t)
	books, _ := newCountingServer(t, files)
	book, err := books.books.acquire(t.Context(), files, "", name)
	if err != nil {
		t.Fatal(err)
	}
	books.Close()
	body, err := book.archive.Entry("EPUB/cover.png")
	if err != nil || !bytes.Equal(body, pngBytes) {
		t.Fatalf("entry = %q, %v, want the picture's bytes while the request reads", body, err)
	}
	books.books.release(book)
	if book.archive != nil {
		t.Error("the archive is still kept after the last request gave the book back")
	}
}
