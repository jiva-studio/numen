package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// newCountingServer is a server on the vault dir that counts the files it opens.
func newCountingServer(t *testing.T, dir string) (*BookServer, *atomic.Int64) {
	t.Helper()
	server := NewBookServerIn(dir)
	t.Cleanup(server.Close)
	var opens atomic.Int64
	server.books.openFile = func(path string) (*os.File, error) {
		opens.Add(1)
		return os.Open(path)
	}
	return server, &opens
}

func readEntryFrom(server *BookServer, name, entry string) (*connect.Response[numenv1.ReadBookEntryResponse], error) {
	return server.ReadBookEntry(context.Background(),
		connect.NewRequest(&numenv1.ReadBookEntryRequest{Path: name, Entry: entry}))
}

func TestReadBookEntryOpensTheBookOnceForManyEntries(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	server, opens := newCountingServer(t, dir)
	for range 50 {
		for _, entry := range []string{"EPUB/cover.png", "EPUB/c1.xhtml", "EPUB/c2.xhtml"} {
			if _, err := readEntryFrom(server, name, entry); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := opens.Load(); got != 1 {
		t.Errorf("the file was opened %d times for 150 entries, want 1", got)
	}
}

func TestReadBookEntryOpensTheBookAgainWhenTheFileChanged(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	server, opens := newCountingServer(t, dir)
	if _, err := readEntryFrom(server, name, "EPUB/cover.png"); err != nil {
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
	if err := os.WriteFile(filepath.Join(dir, name), changed, 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := readEntryFrom(server, name, "EPUB/cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(resp.Msg.GetBody(), pngBytes) {
		t.Error("the entry is the picture of the book before the file changed")
	}
	if got := opens.Load(); got != 2 {
		t.Errorf("the file was opened %d times, want 2", got)
	}
}

func TestReadBookEntryHoldsAtMostSomeBooksOpen(t *testing.T) {
	dir, _, _, _ := writeBook(t)
	server, opens := newCountingServer(t, dir)
	for i := range mostOpenBooks + 1 {
		name := filepath.Join(dir, "copy"+string(rune('a'+i))+".epub")
		if err := os.WriteFile(name, createPictureEpub(t), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := readEntryFrom(server, filepath.Base(name), "EPUB/cover.png"); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(server.books.open); got != mostOpenBooks {
		t.Fatalf("%d books are open, want %d", got, mostOpenBooks)
	}
	// The first book was the least recently read and was let go.
	if _, err := readEntryFrom(server, "copya.epub", "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	if got := opens.Load(); got != mostOpenBooks+2 {
		t.Errorf("the files were opened %d times, want %d", got, mostOpenBooks+2)
	}
}

func TestReadBookEntryAnswersConcurrentRequests(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	server, opens := newCountingServer(t, dir)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 20 {
				resp, err := readEntryFrom(server, name, "EPUB/cover.png")
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(resp.Msg.GetBody(), pngBytes) {
					t.Errorf("the entry is %q, want the picture's bytes", resp.Msg.GetBody())
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

func TestCloseClosesTheFilesOfTheBooksHeld(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	server, _ := newCountingServer(t, dir)
	if _, err := readEntryFrom(server, name, "EPUB/cover.png"); err != nil {
		t.Fatal(err)
	}
	file := server.books.open[0].file
	server.Close()
	if _, err := file.Stat(); err == nil {
		t.Error("the file of the book is still open after Close")
	}
	_, err := readEntryFrom(server, name, "EPUB/cover.png")
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v (%v), want unavailable", connect.CodeOf(err), err)
	}
}

func TestCloseLeavesAReadingRequestItsFile(t *testing.T) {
	dir, name, _, _ := writeBook(t)
	server, _ := newCountingServer(t, dir)
	book, err := server.books.take(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	body, err := book.archive.Entry("EPUB/cover.png")
	if err != nil || !bytes.Equal(body, pngBytes) {
		t.Fatalf("entry = %q, %v, want the picture's bytes while the request reads", body, err)
	}
	server.books.release(book)
	if _, err := book.file.Stat(); err == nil {
		t.Error("the file is still open after the last request gave the book back")
	}
}
