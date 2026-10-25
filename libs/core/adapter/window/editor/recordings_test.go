package editor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

// newRecordingClient is the recordings of a window as a client reaches them, over a window whose
// loopback socket is open.
func newRecordingClient(t *testing.T) numenv1connect.RecordingServiceClient {
	t.Helper()
	api, _ := openRecordingWindow(t, nil)
	back, err := Listen(api.server, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { back.Close() })
	api.openPlaying(back)
	server := httptest.NewServer(api.NewHandler(http.NotFoundHandler()))
	t.Cleanup(server.Close)
	return numenv1connect.NewRecordingServiceClient(server.Client(), server.URL)
}

func TestGetRecordingReturnsLoopbackAddress(t *testing.T) {
	client := newRecordingClient(t)

	answer, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: talk}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetMediaType() != "audio/mpeg" {
		t.Errorf("a recording is played as %q, want audio/mpeg", answer.Msg.GetMediaType())
	}
	played, err := http.Get(answer.Msg.GetMediaUrl())
	if err != nil {
		t.Fatal(err)
	}
	defer played.Body.Close()
	body, err := io.ReadAll(played.Body)
	if err != nil || string(body) != sound {
		t.Errorf("the address served %q (%v), want the recording's bytes", body, err)
	}
}

func TestGetRecordingRejectsMissingFile(t *testing.T) {
	client := newRecordingClient(t)

	_, err := client.GetRecording(t.Context(), connect.NewRequest(&v1.GetRecordingRequest{Path: "talks/none.mp3"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a missing file was answered %v, want not found", err)
	}
}
