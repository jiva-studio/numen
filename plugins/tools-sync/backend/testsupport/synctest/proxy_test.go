package synctest_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

func newBackend(t *testing.T) (*synctest.Server, *atomic.Int32) {
	t.Helper()
	var ran atomic.Int32
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran.Add(1)
		if _, err := io.WriteString(w, "answer to "+r.URL.Path); err != nil {
			t.Error(err)
		}
	})
	return synctest.NewProxy(t, backend), &ran
}

func TestProxy_CountsTheCallsOfEachPath(t *testing.T) {
	server, _ := newBackend(t)

	getStatus(t, server.URL+"/a")
	getStatus(t, server.URL+"/a")
	getStatus(t, server.URL+"/b")

	if server.CountCalls("/a") != 2 || server.CountCalls("/b") != 1 || server.CountRequests() != 3 {
		t.Errorf("calls of /a = %d, of /b = %d, requests = %d; want 2, 1 and 3", server.CountCalls("/a"), server.CountCalls("/b"), server.CountRequests())
	}
}

func TestProxy_AnswersWithoutTheBackendWhileItIsDown(t *testing.T) {
	server, ran := newBackend(t)
	server.SetDown(true)

	status := getStatus(t, server.URL+"/a")

	if status != http.StatusServiceUnavailable || ran.Load() != 0 {
		t.Errorf("status = %d, backend runs = %d; want 503 and no run", status, ran.Load())
	}
}

func TestProxy_RunsTheBackendBeforeItLosesTheResponse(t *testing.T) {
	server, ran := newBackend(t)
	server.LoseNextResponse("/a")

	status := getStatus(t, server.URL+"/a")

	if status != http.StatusServiceUnavailable || ran.Load() != 1 {
		t.Errorf("status = %d, backend runs = %d; want 503 after one run", status, ran.Load())
	}
}

func TestProxy_RunsTheHookOnceAfterTheBackendAnswered(t *testing.T) {
	server, ran := newBackend(t)
	var seen []int32
	server.RunAfterNext("/a", func() { seen = append(seen, ran.Load()) })

	getStatus(t, server.URL+"/a")
	getStatus(t, server.URL+"/a")

	if len(seen) != 1 || seen[0] != 1 {
		t.Errorf("backend runs at each hook = %v, want one hook after the first run", seen)
	}
}

func TestProxy_SendsTheRequestForThePublicAddressToItself(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusAccepted) }))
	t.Cleanup(backend.Close)
	server := synctest.NewRealServer(t, &synctest.Stack{URL: backend.URL, Roots: nil})

	computer := synctest.NewPendingComputer(t, server, synctest.NewClock(synctest.GetStart()), "Test", synctest.NewWords(t))
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, backend.URL+"/a", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := computer.NewHTTP().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusAccepted || server.CountCalls("/a") != 1 {
		t.Errorf("status = %d, calls counted by the proxy = %d; want 202 and 1", response.StatusCode, server.CountCalls("/a"))
	}
}
