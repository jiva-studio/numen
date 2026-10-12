package flashcards

import (
	"github.com/jiva-studio/numen/modules/libs/core/usecase/flashcards"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// A listener that has not read what it was told already is passed over rather
// than waited for: every message says the same thing, so what is said to a
// full listener is what the one it holds already says.
func TestAListenerIsPassedOverRatherThanWaitedFor(t *testing.T) {
	var f changeSubscribers
	line, done := f.subscribe()
	t.Cleanup(done)

	f.notify()
	f.notify()

	if _, is := <-line; !is {
		t.Fatal("the listener was told nothing")
	}
	select {
	case <-line:
		t.Error("the listener holds two of one message")
	default:
	}
}

// A listener that has stopped listening is not spoken to, and does not hold up
// the ones that still are.
func TestAListenerThatStoppedIsNotSpokenTo(t *testing.T) {
	var f changeSubscribers
	going, still := f.subscribe()
	stays, done := f.subscribe()
	t.Cleanup(done)

	still()
	if _, is := <-going; is {
		t.Error("a listener that stopped was still told something")
	}

	f.notify()
	if _, is := <-stays; !is {
		t.Error("the listener that stayed was told nothing")
	}
}

// Something moving underneath the window reaches the page, so a deck written
// while a person is looking at the list is a deck they are shown.
func TestSomethingMovingReachesThePage(t *testing.T) {
	api := newAPIOver(registry{})
	moved := make(chan struct{}, 1)
	api.ReportChangesFrom(t.Context(), moved)

	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)

	stream, err := client.WatchReloads(t.Context(), connect.NewRequest(&v1.WatchReloadsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stream.Close() })

	// The stream says it is listening before anything has moved.
	if !stream.Receive() {
		t.Fatalf("the stream said nothing: %v", stream.Err())
	}

	moved <- struct{}{}

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
	api, held := newAPI(t, deck)
	v := held[0]
	faces := api.CardsDue.CardFaces()

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

	api.ReportChange()
	fresh, err := faces.Execute(t.Context(), v)
	if err != nil || !isFront(fresh[0], "Mould leaf") {
		t.Errorf("the faces stand as they were after a report: %+v %v", fresh, err)
	}
}

func isFront(f flashcards.CardFace, word string) bool {
	front, _ := f.Lay()
	return strings.Contains(front, word)
}
