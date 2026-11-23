package synctest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

// Server is the fake sync server of the protocol library on a loopback port, behind a switch that makes it
// unreachable, a hook that loses one response, and a hook that damages the blobs it sends.
type Server struct {
	Fake *fakeserver.Server
	URL  string

	mu          sync.Mutex
	isDown      bool
	lostPath    string
	isCorrupted bool
}

// NewServer starts the server on the clock and stops it when the test ends.
func NewServer(t *testing.T, clock *Clock, cfg fakeserver.Config) *Server {
	t.Helper()
	cfg.Now = clock.Now
	s := &Server{Fake: fakeserver.New(cfg), URL: "", mu: sync.Mutex{}, isDown: false, lostPath: "", isCorrupted: false}
	served := httptest.NewServer(http.HandlerFunc(s.serve))
	t.Cleanup(served.Close)
	s.URL = served.URL
	s.Fake.SetAudience(served.URL)
	return s
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

// CorruptBlobs makes the server flip the first byte of every blob it sends while isOn is true.
func (s *Server) CorruptBlobs(isOn bool) {
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

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	isDown, isCorrupted := s.isDown, s.isCorrupted
	isLost := s.lostPath != "" && strings.HasSuffix(r.URL.Path, s.lostPath)
	if isLost {
		s.lostPath = ""
	}
	s.mu.Unlock()
	switch {
	case isDown:
		http.Error(w, "the server is down", http.StatusServiceUnavailable)
	case isLost:
		s.Fake.ServeHTTP(httptest.NewRecorder(), r)
		http.Error(w, "the response was lost", http.StatusServiceUnavailable)
	case isCorrupted && r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/blobs/"):
		recorded := httptest.NewRecorder()
		s.Fake.ServeHTTP(recorded, r)
		body := recorded.Body.Bytes()
		if len(body) > 0 {
			body[0] ^= 0xff
		}
		w.WriteHeader(recorded.Code)
		if _, err := w.Write(body); err != nil {
			return
		}
	default:
		s.Fake.ServeHTTP(w, r)
	}
}

// SetUsedBytes sets the bytes the fake counts against the quota of every account, as the admin path of the fake does.
func (s *Server) SetUsedBytes(t *testing.T, bytes uint64) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, s.URL+"/admin/usage?bytes="+strconv.FormatUint(bytes, 10), nil)
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
	if response.StatusCode >= 300 {
		t.Fatalf("the usage admin path answered %d", response.StatusCode)
	}
}
