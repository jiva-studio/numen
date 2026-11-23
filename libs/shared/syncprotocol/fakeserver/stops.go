package fakeserver

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// stopAfter counts the bytes of one direction since the admin call that armed it.
type stopAfter struct {
	isArmed bool
	left    int64
}

func (a *stopAfter) start(bytes int64) {
	a.isArmed, a.left = true, bytes
}

func (a *stopAfter) take(bytes int64) (allowed int64, isEnd bool) {
	if !a.isArmed {
		return bytes, false
	}
	if bytes < a.left {
		a.left -= bytes
		return bytes, false
	}
	allowed = a.left
	a.isArmed, a.left = false, 0
	return allowed, true
}

func (s *Server) countReceivedBytes(bytes int64) {
	if _, isEnd := s.stopUpload.take(bytes); isEnd {
		s.requestStop()
	}
}

func (s *Server) requestStop() {
	if s.stop != nil {
		s.stop()
	}
}

func (s *Server) serveStopAfter(direction *stopAfter) http.HandlerFunc {
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
		direction.start(bytes)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) sendBody(w http.ResponseWriter, r *http.Request, id string, body []byte) {
	from, status := int64(0), http.StatusOK
	if header := r.Header.Get("Range"); header != "" {
		text, found := strings.CutPrefix(header, "bytes=")
		start, isOpen := strings.CutSuffix(text, "-")
		parsed, err := strconv.ParseInt(start, 10, 64)
		if !found || !isOpen || err != nil || parsed < 0 || parsed > int64(len(body)) {
			http.Error(w, "the range cannot be served", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		from, status = parsed, http.StatusPartialContent
	}
	rest := body[from:]
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(rest)))
	if status == http.StatusPartialContent {
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(from, 10)+"-"+strconv.Itoa(len(body)-1)+"/"+strconv.Itoa(len(body)))
	}
	s.mu.Lock()
	allowed, isEnd := s.stopDownload.take(int64(len(rest)))
	s.logDownload(id, from, allowed)
	s.mu.Unlock()
	w.WriteHeader(status)
	if _, err := w.Write(rest[:allowed]); err != nil {
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
