package flashcards

import (
	"net/http"
	"net/http/httptest"
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
	var f following
	line, done := f.listen()
	t.Cleanup(done)

	f.say()
	f.say()

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
	var f following
	going, still := f.listen()
	stays, done := f.listen()
	t.Cleanup(done)

	still()
	if _, is := <-going; is {
		t.Error("a listener that stopped was still told something")
	}

	f.say()
	if _, is := <-stays; !is {
		t.Error("the listener that stayed was told nothing")
	}
}

// Something moving underneath the window reaches the page, so a deck written
// while a person is looking at the list is a deck they are shown.
func TestSomethingMovingReachesThePage(t *testing.T) {
	api := &API{Registry: registry{}, Now: time.Now}
	moved := make(chan struct{}, 1)
	api.Follow(t.Context(), moved)

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
