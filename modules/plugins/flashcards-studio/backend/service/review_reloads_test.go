package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/plugins/flashcards-studio/backend/flashcards"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// Something moving underneath the window reaches the page, so a deck written
// while a person is looking at the list is a deck they are shown.
func TestSomethingMovingReachesThePage(t *testing.T) {
	api, _, installation := newAPIOn(t)
	client := newFlashcardsClient(t, api)

	stream, err := client.WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	// The stream says it is listening before anything has moved.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}

	installation.Host.Move()

	said := make(chan bool, 1)
	go func() { said <- stream.Receive() }()
	select {
	case got := <-said:
		if !got {
			t.Fatalf("the stream ended: %v", stream.Err())
		}
		if !stream.Msg().GetReload() {
			t.Error("the page was not told to ask again")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("something moved and the page was never told")
	}
}

// A deck edited in place keeps its size and its stamp, and the window keeps no
// face of it past a report that something moved.
func TestAReportedChangeForgetsTheFacesAlreadyParsed(t *testing.T) {
	api, held, installation := newAPIOn(t, deck)
	v := held[0]
	faces := api.scenarios().flash.CardsDue.CardFaces()

	before, err := faces.Execute(t.Context(), v)
	if err != nil || len(before) != 1 {
		t.Fatalf("the vault stands as %d card faces and %v", len(before), err)
	}

	at := filepath.Join(v.Path, "decks", "Words.md")
	info, err := os.Stat(at)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(deck["decks/Words.md"], "### Word\n\nLeaf mould", "### Word\n\nMould leaf", 1)
	if err := os.WriteFile(at, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(at, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	stale, err := faces.Execute(t.Context(), v)
	if err != nil || !isFront(stale[0], "Leaf mould") {
		t.Fatalf("the faces were parsed again without a report: %+v %v", stale, err)
	}

	stream, err := newFlashcardsClient(t, api).WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}
	installation.Host.Move()
	if !stream.Receive() || !stream.Msg().GetReload() {
		t.Fatalf("the page was not told to ask again: %v", stream.Err())
	}
	fresh, err := faces.Execute(t.Context(), v)
	if err != nil || !isFront(fresh[0], "Mould leaf") {
		t.Errorf("the faces stand as they were after a report: %+v %v", fresh, err)
	}
}

func isFront(f flashcards.CardFace, word string) bool {
	front, _ := f.Lay()
	return strings.Contains(front, word)
}
