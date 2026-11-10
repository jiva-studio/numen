package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type pipeResponseWriter struct {
	header      http.Header
	code        int
	headerReady chan struct{}
	once        sync.Once
	pw          *io.PipeWriter
}

func newPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	return &pipeResponseWriter{
		header:      make(http.Header),
		code:        http.StatusOK,
		headerReady: make(chan struct{}),
		pw:          pw,
	}
}

func (w *pipeResponseWriter) Header() http.Header {
	return w.header
}

func (w *pipeResponseWriter) Write(b []byte) (int, error) {
	w.once.Do(func() {
		close(w.headerReady)
	})
	return w.pw.Write(b)
}

// Flush sends the response header, so a handler that streams is heard before it ends.
func (w *pipeResponseWriter) Flush() {
	w.once.Do(func() {
		close(w.headerReady)
	})
}

func (w *pipeResponseWriter) WriteHeader(code int) {
	w.code = code
	w.once.Do(func() {
		close(w.headerReady)
	})
}

// InProcessTransport dispatches HTTP requests directly to an in-memory handler.
type InProcessTransport struct {
	handler http.Handler
}

// NewInProcessTransport wraps an HTTP handler in an in-memory round tripper.
func NewInProcessTransport(handler http.Handler) *InProcessTransport {
	return &InProcessTransport{handler: handler}
}

// RoundTrip executes the HTTP request in-process without network overhead.
func (t *InProcessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	pr, pw := io.Pipe()
	rw := newPipeResponseWriter(pw)
	// The body ends with the error of the request context, whether the handler has written its header or not.
	stop := context.AfterFunc(req.Context(), func() { _ = pw.CloseWithError(req.Context().Err()) })

	go func() {
		defer func() {
			stop()
			rw.once.Do(func() {
				close(rw.headerReady)
			})
			if r := recover(); r != nil {
				_ = pw.CloseWithError(fmt.Errorf("panic in in-process handler: %v", r))
			} else {
				_ = pw.CloseWithError(req.Context().Err())
			}
		}()
		t.handler.ServeHTTP(rw, req)
	}()

	select {
	case <-rw.headerReady:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}

	resp := &http.Response{
		StatusCode: rw.code,
		Status:     fmt.Sprintf("%d %s", rw.code, http.StatusText(rw.code)),
		Header:     rw.header,
		Body:       pr,
		Request:    req,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
	return resp, nil
}
