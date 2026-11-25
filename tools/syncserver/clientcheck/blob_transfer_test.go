package client_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

type machineClock struct{}

func (machineClock) Now() time.Time { return time.Now() }

type transferRecord struct {
	method string
	header http.Header
	body   []byte
	length int64
}

type storage struct {
	mu       sync.Mutex
	requests []transferRecord
	stored   []byte
	status   int
}

func (s *storage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, transferRecord{method: r.Method, header: r.Header.Clone(), body: body, length: r.ContentLength})
	if s.status != 0 {
		w.WriteHeader(s.status)
		return
	}
	if r.Method == http.MethodPut {
		s.stored = body
		w.WriteHeader(http.StatusOK)
		return
	}
	if _, err := w.Write(s.stored); err != nil {
		return
	}
}

func serveStorage(tb testing.TB) (*storage, string) {
	tb.Helper()
	store := &storage{}
	server := httptest.NewServer(store)
	tb.Cleanup(server.Close)
	return store, server.URL + "/object"
}

func TestUploadBlob_SendsTheBodyWithItsLengthAndNoBearerOrVersionHeader(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	body := bytes.Repeat([]byte{0x5A}, 4168)

	if err := syncclient.PutBlob(t.Context(), http.DefaultClient, url, body); err != nil {
		t.Fatal(err)
	}

	if len(store.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(store.requests))
	}
	got := store.requests[0]
	if got.method != http.MethodPut || !bytes.Equal(got.body, body) || got.length != int64(len(body)) {
		t.Errorf("request = %s with %d bytes (length %d), want a PUT of the body with its length", got.method, len(got.body), got.length)
	}
	if got.header.Get("Authorization") != "" || got.header.Get(syncprotocol.HeaderVersion) != "" {
		t.Errorf("headers = %v, want no bearer and no protocol version", got.header)
	}
}

func TestGetBlob_ReturnsTheBodyAndSendsNoBearerOrVersionHeader(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	store.stored = []byte("stored object")

	got, err := syncclient.GetBlob(t.Context(), http.DefaultClient, url)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "stored object" {
		t.Errorf("body = %q, want the stored object", got)
	}
	header := store.requests[0].header
	if store.requests[0].method != http.MethodGet || header.Get("Authorization") != "" || header.Get(syncprotocol.HeaderVersion) != "" {
		t.Errorf("request = %s %v, want a GET with no bearer and no protocol version", store.requests[0].method, header)
	}
}

func TestUploadBlobAndDownloadBlob_ReportAnAnswerThatIsNotASuccess(t *testing.T) {
	t.Parallel()
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		store, url := serveStorage(t)
		store.status = status

		if err := syncclient.PutBlob(t.Context(), http.DefaultClient, url, []byte("x")); err == nil {
			t.Errorf("PutBlob with status %d: no error", status)
		}
		if body, err := syncclient.GetBlob(t.Context(), http.DefaultClient, url); err == nil {
			t.Errorf("GetBlob with status %d: body %q and no error", status, body)
		}
	}
}

func TestUploadBlobAndDownloadBlob_StopWhenTheContextEnds(t *testing.T) {
	t.Parallel()
	_, url := serveStorage(t)
	canceled, cancel := context.WithCancel(t.Context())
	cancel()

	if err := syncclient.PutBlob(canceled, http.DefaultClient, url, []byte("x")); err == nil {
		t.Error("PutBlob ran with a context that is done")
	}
	if _, err := syncclient.GetBlob(canceled, http.DefaultClient, url); err == nil {
		t.Error("GetBlob ran with a context that is done")
	}
}

func TestGetChainReadChallenge_ReturnsTheIDAndTheChainReadMessageOfTheRootKey(t *testing.T) {
	t.Parallel()
	server := fakeserver.New(fakeserver.Config{})
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	server.SetAudience(served.URL)
	client := syncclient.NewSignInClient(served.URL, http.DefaultClient, machineClock{}.Now)
	root := bytes.Repeat([]byte{3}, 32)

	read, err := client.GetChainReadChallenge(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	signIn, err := client.GetChallenge(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}

	label := []byte("numen-sync/chain-read/v1\x00")
	tail := binary.BigEndian.AppendUint32(bytes.Clone(root), syncprotocol.Version)
	if len(read.ID) != 16 || !bytes.HasPrefix(read.Message, label) || !bytes.HasSuffix(read.Message, tail) {
		t.Errorf("challenge = %x %x, want a 16-byte id and a message of the chain-read label ending with the root key and the version", read.ID, read.Message)
	}
	if bytes.Equal(read.ID, signIn.ID) {
		t.Error("two challenges have the same id")
	}
}

func TestGetChainReadChallenge_RefusesAChallengeOfAnotherServer(t *testing.T) {
	t.Parallel()
	server := fakeserver.New(fakeserver.Config{Audience: "https://other.example"})
	served := httptest.NewServer(server)
	t.Cleanup(served.Close)
	client := syncclient.NewSignInClient(served.URL, http.DefaultClient, machineClock{}.Now)

	_, err := client.GetChainReadChallenge(t.Context(), bytes.Repeat([]byte{3}, 32))

	if !errors.Is(err, syncprotocol.ErrAudienceMismatch) {
		t.Errorf("error = %v, want ErrAudienceMismatch", err)
	}
}

type rangeStorage struct {
	mu       sync.Mutex
	requests []transferRecord
	object   []byte
	ignores  bool
}

func (s *rangeStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, transferRecord{method: r.Method, header: r.Header.Clone()})
	s.mu.Unlock()
	if r.Method == http.MethodPut {
		w.WriteHeader(http.StatusOK)
		return
	}
	if s.ignores {
		if _, err := w.Write(s.object); err != nil {
			return
		}
		return
	}
	http.ServeContent(w, r, "object", time.Time{}, bytes.NewReader(s.object))
}

func serveRangeStorage(tb testing.TB, object []byte, ignoresRange bool) (*rangeStorage, string) {
	tb.Helper()
	store := &rangeStorage{object: object, ignores: ignoresRange}
	server := httptest.NewServer(store)
	tb.Cleanup(server.Close)
	return store, server.URL + "/object"
}

func TestUploadPart_SendsTheBodyWithItsLengthAndNoBearerOrVersionHeader(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	body := bytes.Repeat([]byte{0x33}, 5<<20)

	if err := syncclient.PutPart(t.Context(), http.DefaultClient, url, body); err != nil {
		t.Fatal(err)
	}

	got := store.requests[0]
	if got.method != http.MethodPut || !bytes.Equal(got.body, body) || got.length != int64(len(body)) {
		t.Errorf("request = %s with %d bytes (length %d), want a PUT of the part with its length", got.method, len(got.body), got.length)
	}
	if got.header.Get("Authorization") != "" || got.header.Get(syncprotocol.HeaderVersion) != "" {
		t.Errorf("headers = %v, want no bearer and no protocol version", got.header)
	}
}

func TestUploadPart_ReportsAnAnswerThatIsNotASuccess(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	store.status = http.StatusForbidden

	if err := syncclient.PutPart(t.Context(), http.DefaultClient, url, []byte("part")); err == nil {
		t.Error("a 403 gave no error")
	}
}

func TestGetBlobRange_StreamsTheBodyFromTheGivenByteWithARangeHeaderAndNoBearer(t *testing.T) {
	t.Parallel()
	object := bytes.Repeat([]byte("0123456789"), 100)
	store, url := serveRangeStorage(t, object, false)

	body, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, url, 250)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(body)
	if closeErr := body.Close(); err != nil || closeErr != nil {
		t.Fatalf("read: %v, close: %v", err, closeErr)
	}

	if !bytes.Equal(got, object[250:]) {
		t.Errorf("read %d bytes, want the %d bytes from byte 250", len(got), len(object)-250)
	}
	header := store.requests[0].header
	if header.Get("Range") != "bytes=250-" || header.Get("Authorization") != "" || header.Get(syncprotocol.HeaderVersion) != "" {
		t.Errorf("headers = %v, want Range bytes=250- and no bearer and no protocol version", header)
	}
}

func TestGetBlobRange_ReturnsTheWholeBodyForFromZeroEvenWhenTheStorageIgnoresRange(t *testing.T) {
	t.Parallel()
	object := []byte("whole object")
	_, url := serveRangeStorage(t, object, true)

	body, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, url, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(body)

	if err != nil || !bytes.Equal(got, object) {
		t.Errorf("read %q, error = %v, want the whole object", got, err)
	}
}

func TestGetBlobRange_RefusesAFullBodyForAStartAfterZero(t *testing.T) {
	t.Parallel()
	_, url := serveRangeStorage(t, []byte("whole object"), true)

	body, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, url, 5)

	if err == nil {
		if closeErr := body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Error("a 200 answer to a ranged request from byte 5 gave no error")
	}
}

func TestGetBlobRange_ReportsAnAnswerThatIsNotASuccessAndAnEndedContext(t *testing.T) {
	t.Parallel()
	store, url := serveStorage(t)
	store.status = http.StatusForbidden

	if _, err := syncclient.GetBlobRange(t.Context(), http.DefaultClient, url, 0); err == nil {
		t.Error("a 403 gave no error")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := syncclient.GetBlobRange(canceled, http.DefaultClient, url, 0); err == nil {
		t.Error("a context that is done gave no error")
	}
}
