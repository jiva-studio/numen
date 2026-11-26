package synctest

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type PartLogEntry struct {
	Blob      string
	Part      uint32
	Direction string
	From      int64
	Bytes     int64
}

// blobEdge is the network edge of the computers towards the bucket that holds the blobs: the signed addresses of the real server lead there,
// not to the server, so the faults of the blob transfers are made on the side of the plugin, in the link of each computer.
type blobEdge struct {
	mu           sync.Mutex
	gets         int
	budget       int
	hasBudget    bool
	isCorrupted  bool
	uploadLimit  int64
	hasUpLimit   bool
	downLimit    int64
	hasDownLimit bool
	log          []PartLogEntry
}

func (s *Server) CountBlobGets() int {
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	return s.blob.gets
}

// FailBlobGetsAfter makes every download of a blob answer 503 once n more have been started, as a connection that
// drops in the middle of a file does. A negative n lifts the limit.
func (s *Server) FailBlobGetsAfter(n int) {
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	s.blob.hasBudget, s.blob.budget = n >= 0, n
}

func (s *Server) SetBlobCorruption(isOn bool) {
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	s.blob.isCorrupted = isOn
}

func (s *Server) StopAfterUploadBytes(t *testing.T, n int64) {
	t.Helper()
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	s.blob.hasUpLimit, s.blob.uploadLimit = true, n
}

func (s *Server) StopAfterDownloadBytes(t *testing.T, n int64) {
	t.Helper()
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	s.blob.hasDownLimit, s.blob.downLimit = true, n
}

func (s *Server) GetPartLog(t *testing.T) []PartLogEntry {
	t.Helper()
	s.blob.mu.Lock()
	defer s.blob.mu.Unlock()
	return append([]PartLogEntry(nil), s.blob.log...)
}

func (s *Server) roundTripBlob(r *http.Request) (*http.Response, error) {
	if s.isUnreachable() {
		return refuseBlob(r), nil
	}
	if r.Method == http.MethodGet && s.blob.takeGet() {
		return refuseBlob(r), nil
	}
	if r.Method == http.MethodPut {
		return s.putBlob(r)
	}
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil || r.Method != http.MethodGet {
		return response, err
	}
	return s.readBlob(r, response)
}

func (s *Server) isUnreachable() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isDown
}

func (e *blobEdge) takeGet() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gets++
	if !e.hasBudget {
		return false
	}
	if e.budget <= 0 {
		return true
	}
	e.budget--
	return false
}

func (s *Server) putBlob(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		return nil, err
	}
	size := max(r.ContentLength, 0)
	part, err := strconv.ParseUint(r.URL.Query().Get("partNumber"), 10, 32)
	if err != nil {
		part = 0
	}
	s.blob.mu.Lock()
	s.blob.log = append(s.blob.log, PartLogEntry{Blob: getBlobName(r), Part: uint32(part), Direction: "put", From: 0, Bytes: size})
	isStopped := false
	if s.blob.hasUpLimit {
		s.blob.uploadLimit -= size
		isStopped = s.blob.uploadLimit <= 0
		s.blob.hasUpLimit = !isStopped
	}
	s.blob.mu.Unlock()
	if isStopped {
		s.SetDown(true)
	}
	return response, nil
}

func (s *Server) readBlob(r *http.Request, response *http.Response) (*http.Response, error) {
	body, err := io.ReadAll(response.Body)
	if closeErr := response.Body.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	s.blob.mu.Lock()
	if s.blob.isCorrupted && len(body) > 0 {
		body[0] ^= 0xff
	}
	from := int64(0)
	if _, after, isRange := strings.Cut(r.Header.Get("Range"), "="); isRange {
		first, _, _ := strings.Cut(after, "-")
		parsed, err := strconv.ParseInt(first, 10, 64)
		if err == nil {
			from = parsed
		}
	}
	direction := "get"
	if r.Header.Get("Range") != "" {
		direction = "get-range"
	}
	s.blob.log = append(s.blob.log, PartLogEntry{Blob: getBlobName(r), Part: 0, Direction: direction, From: from, Bytes: int64(len(body))})
	isStopped := false
	delivered := body
	if s.blob.hasDownLimit {
		if s.blob.downLimit < int64(len(body)) {
			delivered = body[:max(s.blob.downLimit, 0)]
		}
		s.blob.downLimit -= int64(len(body))
		isStopped = s.blob.downLimit <= 0
		s.blob.hasDownLimit = !isStopped
	}
	s.blob.mu.Unlock()
	if isStopped {
		s.SetDown(true)
	}
	if len(delivered) < len(body) {
		response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(delivered), errReader{io.ErrUnexpectedEOF}))
		return response, nil
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	response.ContentLength = int64(len(body))
	response.Header.Del("Content-Length")
	return response, nil
}

func getBlobName(r *http.Request) string {
	return strings.TrimPrefix(r.URL.Path, "/")
}

func refuseBlob(r *http.Request) *http.Response {
	return &http.Response{
		Status: "503 Service Unavailable", StatusCode: http.StatusServiceUnavailable, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
		Header: http.Header{}, Body: io.NopCloser(strings.NewReader("")), Request: r, ContentLength: 0,
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }
