package client_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
)

func TestPutPart_SendsTheBodyWithItsLengthAndNoBearerOrVersionHeader(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	body := bytes.Repeat([]byte{0x33}, 5<<20)

	if err := syncclient.PutPart(t.Context(), http.DefaultClient, url, body); err != nil {
		t.Fatal(err)
	}

	got := store.requests[0]
	if got.method != http.MethodPut || got.length != int64(len(body)) || got.header.Get("Authorization") != "" || got.header.Get(syncprotocol.HeaderVersion) != "" {
		t.Errorf("request = %s of length %d with headers %v, want a PUT with the length and no bearer", got.method, got.length, got.header)
	}
}

func TestPutPart_ReportsAnAnswerThatIsNotASuccess(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	store.status = http.StatusForbidden

	if err := syncclient.PutPart(t.Context(), http.DefaultClient, url, []byte("x")); err == nil {
		t.Error("a refused part gave no error")
	}
}

func serveRanges(tb testing.TB, object []byte) string {
	tb.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "object", time.Time{}, bytes.NewReader(object))
	}))
	tb.Cleanup(server.Close)
	return server.URL
}

func TestGetBlobRange_StreamsTheBodyFromTheByteAndAsksForARange(t *testing.T) {
	t.Parallel()
	object := bytes.Repeat([]byte("0123456789"), 100)
	url := serveRanges(t, object)

	for _, from := range []int64{0, 250} {
		stream, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, url, from)
		if err != nil {
			t.Fatalf("from %d: %v", from, err)
		}
		got, err := io.ReadAll(stream)
		if err != nil || !bytes.Equal(got, object[from:]) {
			t.Errorf("from %d: read %d bytes, error = %v, want %d", from, len(got), err, len(object)-int(from))
		}
		if err := stream.Close(); err != nil {
			t.Error(err)
		}
	}
}

func TestGetBlobRange_RefusesAFullBodyWhereARangeWasAskedFor(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte("whole")); err != nil {
			return
		}
	}))
	t.Cleanup(server.Close)

	if _, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, server.URL, 10); err == nil {
		t.Error("a 200 for a range from 10 gave no error")
	}
}
