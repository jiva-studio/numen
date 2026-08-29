package flashcardsui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Something moving underneath the window reaches the page, so a deck written
// while a person is looking at the list is a deck they are shown.
func TestSomethingMovingReachesThePage(t *testing.T) {
	api := &API{Registry: registry{}, Now: time.Now}
	moved := make(chan struct{}, 1)
	api.Follows(t.Context(), moved)

	server := httptest.NewServer(api.Serving(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	client := numenv1connect.NewFlashcardsServiceClient(server.Client(), server.URL)

	stream, err := client.Moving(t.Context(), connect.NewRequest(&v1.MovingRequest{}))
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
