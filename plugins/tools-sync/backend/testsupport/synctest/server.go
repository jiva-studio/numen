package synctest

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

// Server is the fake sync server of the protocol library on a loopback port, behind a switch that makes it
// unreachable, a hook that loses one response, and a hook that damages the blobs it sends.
// Over the real sync server it is a proxy of the test in front of it, and URL is then the public address of that server, which the computers dial and the server names as its own.
type Server struct {
	Fake *fakeserver.Server
	URL  string

	backend      http.Handler
	dial         string
	origin       string
	roots        []ed25519.PublicKey
	calls        map[string]int
	mu           sync.Mutex
	isDown       bool
	lostPath     string
	isCorrupted  bool
	hookPath     string
	hookCallback func()
	requests     int
	blobGets     int
	blobBudget   int
	hasBudget    bool
}

// NewServer starts the server on the clock and stops it when the test ends. An admin stop budget that ends makes the
// server unreachable, as a program that exits does, until SetDown(false).
func NewServer(t *testing.T, clock *Clock, cfg fakeserver.Config) *Server {
	t.Helper()
	cfg.Now = clock.Now
	s := newServer()
	cfg.Stop = func() { s.SetDown(true) }
	s.Fake = fakeserver.New(cfg)
	s.backend = s.Fake
	served := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(served.Close)
	s.URL = served.URL
	s.Fake.SetAudience(served.URL)
	return s
}

// PartLogEntry is a row of the part log of the server: a part received or a range of an object sent.
type PartLogEntry struct {
	Blob      string `json:"blob"`
	Part      uint32 `json:"part"`
	Direction string `json:"direction"`
	From      int64  `json:"from"`
	Bytes     int64  `json:"bytes"`
}

// StopAfterUploadBytes makes the server unreachable once it received n bytes of parts from now on.
func (s *Server) StopAfterUploadBytes(t *testing.T, n int64) {
	t.Helper()
	s.callAdmin(t, fmt.Sprintf("/admin/stop-after-bytes?n=%d", n))
}

// StopAfterDownloadBytes makes the server unreachable once it sent n bytes of blobs from now on.
func (s *Server) StopAfterDownloadBytes(t *testing.T, n int64) {
	t.Helper()
	s.callAdmin(t, fmt.Sprintf("/admin/stop-after-download-bytes?n=%d", n))
}

func (s *Server) callAdmin(t *testing.T, path string) {
	t.Helper()
	if status := s.CallAdmin(t, path); status != http.StatusNoContent {
		t.Fatalf("%s answered %d, want 204", path, status)
	}
}

// CallAdmin sends a POST to an admin path of the fake server and returns the status.
func (s *Server) CallAdmin(t *testing.T, path string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Error(err)
	}
	return response.StatusCode
}

// SetUsedBytes sets the bytes the fake counts against the quota of every account, as the admin path of the fake does.
func (s *Server) SetUsedBytes(t *testing.T, bytes uint64) {
	t.Helper()
	s.callAdmin(t, fmt.Sprintf("/admin/usage?bytes=%d", bytes))
}

// GetPartLog returns the parts the server received and the ranges of objects it sent, in order.
func (s *Server) GetPartLog(t *testing.T) []PartLogEntry {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL+"/admin/part-log", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	var logged []PartLogEntry
	if err := json.NewDecoder(response.Body).Decode(&logged); err != nil {
		t.Fatal(err)
	}
	return logged
}

// SetDown makes every request fail with 503 while isDown is true.
func (s *Server) SetDown(isDown bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isDown = isDown
}

// LoseNextResponse runs the next request whose path ends with the suffix on the server and answers 503, as a
// connection that breaks after the server ran the call does.
func (s *Server) LoseNextResponse(suffix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lostPath = suffix
}

// SetBlobCorruption makes the server flip the first byte of every blob it sends while isOn is true.
func (s *Server) SetBlobCorruption(isOn bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isCorrupted = isOn
}

// GetStoredBlobs returns what the server holds of every blob, as the admin path of the fake serves it.
func (s *Server) GetStoredBlobs(t *testing.T) []byte {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL+"/admin/blobs", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// RunAfterNext runs the hook once the server has computed its answer to the next request whose path ends with the
// suffix and before the caller gets it, as another computer does that acts between two calls of a pass.
func (s *Server) RunAfterNext(suffix string, hook func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hookPath, s.hookCallback = suffix, hook
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	isDown, isCorrupted, isLost, hook := s.takeFaults(r.Method, r.URL.Path)
	switch {
	case isDown:
		http.Error(w, "the server is down", http.StatusServiceUnavailable)
	case isLost:
		s.backend.ServeHTTP(httptest.NewRecorder(), r)
		http.Error(w, "the response was lost", http.StatusServiceUnavailable)
	case isCorrupted && r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/blobs/"):
		s.serveCorrupted(w, r)
	case hook != nil:
		s.serveHooked(w, r, hook)
	default:
		s.backend.ServeHTTP(w, r)
	}
}

// ForgetRemovals makes the server drop every removal it keeps, as it does when they are older than its
// retention.
func (s *Server) ForgetRemovals() { s.Fake.PruneTombstones() }

func (s *Server) takeFaults(method, path string) (isDown, isCorrupted, isLost bool, hook func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[path]++
	isCut := s.countRequest(method, path)
	isLost = !s.isDown && s.lostPath != "" && strings.HasSuffix(path, s.lostPath)
	if isLost {
		s.lostPath = ""
	}
	if s.hookPath != "" && strings.HasSuffix(path, s.hookPath) {
		hook, s.hookPath, s.hookCallback = s.hookCallback, "", nil
	}
	return s.isDown || isCut, s.isCorrupted, isLost, hook
}

func (s *Server) countRequest(method, path string) bool {
	s.requests++
	if method != http.MethodGet || !strings.HasPrefix(path, "/blobs/") {
		return false
	}
	s.blobGets++
	if !s.hasBudget {
		return false
	}
	if s.blobBudget <= 0 {
		return true
	}
	s.blobBudget--
	return false
}

// CountRequests returns how many requests the server has taken, answered or not.
func (s *Server) CountRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

// CountBlobGets returns how many downloads of a blob, whole or in a range, the server has taken.
func (s *Server) CountBlobGets() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.blobGets
}

// FailBlobGetsAfter makes every download of a blob answer 503 once n more have been taken, as a connection that
// drops in the middle of a file does. A negative n lifts the limit.
func (s *Server) FailBlobGetsAfter(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hasBudget, s.blobBudget = n >= 0, n
}

func (s *Server) serveCorrupted(w http.ResponseWriter, r *http.Request) {
	recorded := httptest.NewRecorder()
	s.backend.ServeHTTP(recorded, r)
	body := recorded.Body.Bytes()
	if len(body) > 0 {
		body[0] ^= 0xff
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(recorded.Code)
	if _, err := w.Write(body); err != nil {
		return
	}
}

func (s *Server) serveHooked(w http.ResponseWriter, r *http.Request, hook func()) {
	recorded := httptest.NewRecorder()
	s.backend.ServeHTTP(recorded, r)
	hook()
	maps.Copy(w.Header(), recorded.Header())
	switch replayed := recorded.Header().Get("Content-Type"); {
	case strings.HasPrefix(replayed, "application/proto"):
		w.Header().Set("Content-Type", "application/proto")
	case strings.HasPrefix(replayed, "application/connect+proto"):
		w.Header().Set("Content-Type", "application/connect+proto")
	case strings.HasPrefix(replayed, "application/json"):
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(recorded.Code)
	if _, err := w.Write(recorded.Body.Bytes()); err != nil {
		return
	}
}

func newServer() *Server {
	return &Server{
		Fake: nil, URL: "", backend: nil, dial: "", origin: "", roots: nil, calls: map[string]int{}, mu: sync.Mutex{}, isDown: false,
		lostPath: "", isCorrupted: false, hookPath: "", hookCallback: nil, requests: 0, blobGets: 0, blobBudget: 0, hasBudget: false,
	}
}

func fakeRoots() []ed25519.PublicKey { return fakeserver.GetTestRoots() }
