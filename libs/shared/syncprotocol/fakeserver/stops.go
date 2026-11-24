package fakeserver

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/stopafter"
)

func (s *Server) countReceivedBytes(bytes int64) {
	if _, isEnd := s.stopUpload.Take(bytes); isEnd {
		s.requestStop()
	}
}

func (s *Server) requestStop() {
	if s.stop != nil {
		s.stop()
	}
}

func (s *Server) serveStopAfter(direction *stopafter.Limit) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		bytes, err := strconv.ParseInt(r.URL.Query().Get("n"), 10, 64)
		if err != nil || bytes < 0 {
			http.Error(w, "n is not a count of bytes", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		direction.Start(bytes)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) sendBody(w http.ResponseWriter, r *http.Request, id string, body io.ReadSeeker, size int64) {
	from, status := int64(0), http.StatusOK
	if header := r.Header.Get("Range"); header != "" {
		text, found := strings.CutPrefix(header, "bytes=")
		start, isOpen := strings.CutSuffix(text, "-")
		parsed, err := strconv.ParseInt(start, 10, 64)
		if !found || !isOpen || err != nil || parsed < 0 || parsed > size {
			http.Error(w, "the range cannot be served", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		from, status = parsed, http.StatusPartialContent
	}
	rest := size - from
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(rest, 10))
	if status == http.StatusPartialContent {
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(from, 10)+"-"+strconv.FormatInt(size-1, 10)+"/"+strconv.FormatInt(size, 10))
	}
	s.mu.Lock()
	allowed, isEnd := s.stopDownload.Take(rest)
	s.logDownload(id, from, allowed)
	s.mu.Unlock()
	w.WriteHeader(status)
	if _, err := body.Seek(from, io.SeekStart); err != nil {
		return
	}
	if _, err := io.CopyN(w, body, allowed); err != nil {
		return
	}
	if isEnd {
		controller := http.NewResponseController(w)
		flushErr := controller.Flush()
		s.requestStop()
		if flushErr != nil {
			return
		}
		conn, _, err := controller.Hijack()
		if err != nil {
			return
		}
		if err := conn.Close(); err != nil {
			slog.Debug("closing the cut download connection", "error", err)
		}
	}
}
