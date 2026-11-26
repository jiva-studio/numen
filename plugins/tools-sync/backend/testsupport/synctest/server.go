package synctest

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1/syncv1connect"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/wiretest"
)

// Server is a proxy of the test in front of the real sync server: a switch that makes it unreachable, a hook that loses one response, a hook that runs after the server answered, and the counters of the calls.
// URL is the public address of the real server, which the computers dial and the server names as its own.
type Server struct {
	URL   string
	Stack *Stack

	backend      http.Handler
	dial         string
	origin       string
	roots        []ed25519.PublicKey
	calls        map[string]int
	mu           sync.Mutex
	isDown       bool
	lostPath     string
	hookPath     string
	hookCallback func()
	requests     int
	protocol     string
	seen         map[string]bool
	ended        map[string]bool
	streams      map[int]context.CancelFunc
	channels     [][]byte
	nextStream   int
	bucketHost   string
	blob         blobEdge
}

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

// RunAfterNext runs the hook once the server has computed its answer to the next request whose path ends with the
// suffix and before the caller gets it, as another computer does that acts between two calls of a pass.
func (s *Server) RunAfterNext(suffix string, hook func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hookPath, s.hookCallback = suffix, hook
}

// CountCalls returns how many requests for the path the server has taken, answered or not, for example
// syncv1connect.BlobServiceReserveBlobsProcedure.
func (s *Server) CountCalls(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[path]
}

func (s *Server) CountRequests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	r = s.takeToken(r)
	r, closeStream := s.openStream(r)
	defer closeStream()
	isDown, isLost, hook := s.takeFaults(r.URL.Path)
	switch {
	case isDown:
		http.Error(w, "the server is down", http.StatusServiceUnavailable)
	case isLost:
		s.backend.ServeHTTP(httptest.NewRecorder(), r)
		http.Error(w, "the response was lost", http.StatusServiceUnavailable)
	case hook != nil:
		s.serveHooked(w, r, hook)
	case strings.HasSuffix(r.URL.Path, syncv1connect.SignInServiceSignInProcedure):
		s.serveSignIn(w, r)
	case strings.Contains(r.URL.Path, ".LinkService/"):
		s.serveLink(w, r)
	default:
		s.backend.ServeHTTP(w, r)
	}
}

func (s *Server) takeFaults(path string) (isDown, isLost bool, hook func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls[path]++
	s.requests++
	isLost = !s.isDown && s.lostPath != "" && strings.HasSuffix(path, s.lostPath)
	if isLost {
		s.lostPath = ""
	}
	if s.hookPath != "" && strings.HasSuffix(path, s.hookPath) {
		hook, s.hookPath, s.hookCallback = s.hookCallback, "", nil
	}
	return s.isDown, isLost, hook
}

func (s *Server) serveHooked(w http.ResponseWriter, r *http.Request, hook func()) {
	recorded := httptest.NewRecorder()
	s.backend.ServeHTTP(recorded, r)
	hook()
	replay(w, recorded)
}

func (s *Server) serveSignIn(w http.ResponseWriter, r *http.Request) {
	recorded := httptest.NewRecorder()
	s.backend.ServeHTTP(recorded, r)
	var opened syncv1.SignInResponse
	if wiretest.DecodeMessage(unzipBody(recorded), &opened) == nil && opened.GetSessionToken() != "" {
		s.mu.Lock()
		s.seen[opened.GetSessionToken()] = true
		s.mu.Unlock()
	}
	replay(w, recorded)
}

func replay(w http.ResponseWriter, recorded *httptest.ResponseRecorder) {
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
		URL: "", Stack: nil, backend: nil, dial: "", origin: "", roots: nil, calls: map[string]int{}, mu: sync.Mutex{}, isDown: false,
		lostPath: "", hookPath: "", hookCallback: nil, requests: 0, protocol: "", seen: map[string]bool{}, ended: map[string]bool{}, streams: map[int]context.CancelFunc{},
		nextStream: 0, channels: nil, bucketHost: "",
		blob: blobEdge{mu: sync.Mutex{}, gets: 0, budget: 0, hasBudget: false, isCorrupted: false, uploadLimit: 0, hasUpLimit: false, downLimit: 0, hasDownLimit: false, log: nil},
	}
}

// EndSessions ends every session the server has given so far: a call that carries one of their tokens reaches the server with a token it does not know and is
// refused as the server refuses a session that has ended. Sessions opened after this call work.
func (s *Server) EndSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for token := range s.seen {
		s.ended[token] = true
	}
}

func (s *Server) EndStreams() {
	s.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(s.streams))
	for _, cancel := range s.streams {
		cancels = append(cancels, cancel)
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (s *Server) CountStreams() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.streams)
}

func (s *Server) isBlobHost(host string) bool { return s.bucketHost != "" && host == s.bucketHost }

func (s *Server) takeToken(r *http.Request) *http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := r
	if s.protocol != "" {
		changed = r.Clone(r.Context())
		changed.Header.Set(syncprotocol.HeaderVersion, s.protocol)
	}
	token, hasToken := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !hasToken {
		return changed
	}
	if !s.ended[token] {
		s.seen[token] = true
		return changed
	}
	if changed == r {
		changed = r.Clone(r.Context())
	}
	changed.Header.Set("Authorization", "Bearer ended")
	return changed
}

func (s *Server) openStream(r *http.Request) (*http.Request, func()) {
	if !strings.HasSuffix(r.URL.Path, "/WatchChanges") {
		return r, func() {}
	}
	ctx, cancel := context.WithCancel(r.Context())
	s.mu.Lock()
	id := s.nextStream
	s.nextStream++
	s.streams[id] = cancel
	s.mu.Unlock()
	return r.WithContext(ctx), func() {
		cancel()
		s.mu.Lock()
		delete(s.streams, id)
		s.mu.Unlock()
	}
}

func (s *Server) SetProtocol(version uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.protocol = strconv.FormatUint(uint64(version), 10)
}

func (s *Server) NewHTTP() *http.Client {
	return &http.Client{Transport: &link{isOffline: atomic.Bool{}, origin: s.origin, dial: s.dial, server: s}}
}

func (s *Server) ReadSession(t *testing.T, token string) (string, bool) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, s.URL+"/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := s.NewHTTP().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return "", false
	}
	var live struct {
		AccountFingerprint string `json:"account_fingerprint"`
	}
	if err := json.NewDecoder(response.Body).Decode(&live); err != nil {
		t.Fatal(err)
	}
	return live.AccountFingerprint, true
}

func (c *Computer) HasSession(t *testing.T) bool {
	t.Helper()
	_, isLive := c.Server.ReadSession(t, c.getToken())
	return isLive
}

func unzipBody(recorded *httptest.ResponseRecorder) []byte {
	body := recorded.Body.Bytes()
	if recorded.Header().Get("Content-Encoding") != "gzip" {
		return body
	}
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	plain, err := io.ReadAll(reader)
	if closeErr := reader.Close(); err != nil || closeErr != nil {
		return nil
	}
	return plain
}

const (
	linkRetries    = 90
	linkRetryLimit = 65 * time.Second
	linkRetryFloor = 10
)

func (s *Server) serveLink(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var recorded *httptest.ResponseRecorder
	for attempt := 0; ; attempt++ {
		again := r.Clone(r.Context())
		again.Body = io.NopCloser(bytes.NewReader(body))
		recorded = httptest.NewRecorder()
		s.backend.ServeHTTP(recorded, again)
		wait, parseErr := strconv.Atoi(recorded.Header().Get("Retry-After"))
		if parseErr != nil {
			wait = 0
		}
		if recorded.Code != http.StatusTooManyRequests || attempt == linkRetries {
			break
		}
		select {
		case <-r.Context().Done():
			replay(w, recorded)
			return
		case <-time.After(min(time.Duration(max(wait, linkRetryFloor))*time.Second, linkRetryLimit)):
		}
	}
	recordChannel(s, r, recorded)
	replay(w, recorded)
}

func recordChannel(s *Server, r *http.Request, recorded *httptest.ResponseRecorder) {
	if !strings.HasSuffix(r.URL.Path, syncv1connect.LinkServiceCreateChannelProcedure) || recorded.Code != http.StatusOK {
		return
	}
	var made syncv1.CreateChannelResponse
	if wiretest.DecodeMessage(unzipBody(recorded), &made) != nil || len(made.GetChannelId()) == 0 {
		return
	}
	s.mu.Lock()
	s.channels = append(s.channels, made.GetChannelId())
	s.mu.Unlock()
}

func (s *Server) closeChannels(t *testing.T) {
	t.Helper()
	s.mu.Lock()
	channels := s.channels
	s.channels = nil
	s.mu.Unlock()
	relay := syncclient.NewLinkClient(s.URL, s.NewHTTP())
	for _, id := range channels {
		if err := relay.CloseChannel(context.WithoutCancel(t.Context()), id); err != nil && !errors.Is(err, syncprotocol.ErrChannelGone) {
			t.Logf("close a channel of the test: %v", err)
		}
	}
}
