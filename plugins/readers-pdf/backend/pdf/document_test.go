package pdf

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

type recordingTimer struct {
	stopped bool
}

func (r *recordingTimer) Stop() bool {
	r.stopped = true
	return true
}

type seekFailCloser struct {
	err             error
	failOnSeekEnd   bool
	failOnSeekStart bool
	isClosed        bool
	lastOffset      int64
}

func (s *seekFailCloser) Read([]byte) (int, error) { return 0, io.EOF }

func (s *seekFailCloser) Seek(offset int64, whence int) (int64, error) {
	s.lastOffset = offset
	if offset != 0 {
		return 0, errors.New("expected zero seek offset")
	}
	if whence == io.SeekEnd && s.failOnSeekEnd {
		return 0, s.err
	}
	if whence == io.SeekStart && s.failOnSeekStart {
		return 0, s.err
	}
	return 100, nil
}

func (s *seekFailCloser) Close() error {
	s.isClosed = true
	return nil
}

func TestOpenReaderClosesFileOnSeekError(t *testing.T) {
	errSeekEnd := errors.New("seek end failed")
	errSeekStart := errors.New("seek start failed")

	tests := []struct {
		name            string
		expectedErr     error
		failOnSeekEnd   bool
		failOnSeekStart bool
	}{
		{
			name:          "seek end failure",
			expectedErr:   errSeekEnd,
			failOnSeekEnd: true,
		},
		{
			name:            "seek start failure",
			expectedErr:     errSeekStart,
			failOnSeekStart: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fc := &seekFailCloser{
				err:             tc.expectedErr,
				failOnSeekEnd:   tc.failOnSeekEnd,
				failOnSeekStart: tc.failOnSeekStart,
			}
			doc, err := newWorkers(t).openReader(fc)
			if err == nil {
				if doc != nil {
					closeAndCheck(t, doc)
				}
				t.Fatal("openReader succeeded, want seek error")
			}
			if !fc.isClosed {
				t.Error("file was not closed on seek failure")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("error %v does not wrap %v", err, tc.expectedErr)
			}
			if unwrapped := errors.Unwrap(err); !errors.Is(unwrapped, tc.expectedErr) {
				t.Errorf("errors.Unwrap(err) = %v, want %v", unwrapped, tc.expectedErr)
			}
		})
	}
}

type trackingCloser struct {
	*bytes.Reader
	isClosed bool
}

func (t *trackingCloser) Close() error {
	t.isClosed = true
	return nil
}

func TestOpenReaderEmptyFileClosesFile(t *testing.T) {
	fc := &trackingCloser{Reader: bytes.NewReader(nil)}
	_, err := newWorkers(t).openReader(fc)
	if !errors.Is(err, ErrNotPDF) {
		t.Errorf("expected ErrNotPDF for empty file, got %v", err)
	}
	if !fc.isClosed {
		t.Error("expected empty file to be closed")
	}
}

func TestOpenReaderInvalidContentClosesFile(t *testing.T) {
	fc := &trackingCloser{Reader: bytes.NewReader([]byte("not a valid pdf content"))}
	_, err := newWorkers(t).openReader(fc)
	if !errors.Is(err, ErrNotPDF) {
		t.Errorf("expected ErrNotPDF for invalid file, got %v", err)
	}
	if !fc.isClosed {
		t.Error("expected invalid file to be closed")
	}
}

func TestDocumentCloseClosesFile(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	fc := &trackingCloser{Reader: bytes.NewReader(raw)}
	doc, err := newWorkers(t).openReader(fc)
	if err != nil {
		t.Fatal(err)
	}
	if fc.isClosed {
		t.Error("file should not be closed while document is open")
	}
	closeAndCheck(t, doc)
	if !fc.isClosed {
		t.Error("doc.Close() did not close file")
	}
}

func TestDocumentTextInvalidPageIndex(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := newWorkers(t).openBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAndCheck(t, doc)

	if got := doc.text(-1); got != "" {
		t.Errorf("doc.text(-1) = %q, want empty string", got)
	}
	if got := doc.text(9999); got != "" {
		t.Errorf("doc.text(9999) = %q, want empty string", got)
	}
}

func TestGetRuntimeConfigAndTimerLifecycle(t *testing.T) {
	if cfg := NewWorkers(Options{Report: failOnReport(t), CompilationCache: nil, AfterFunc: nil, Workers: 0, IdleWorkers: 0}).getRuntimeConfig(); cfg != nil {
		t.Fatalf("expected nil RuntimeConfig when compilation cache is nil, got %v", cfg)
	}

	var timers []*recordingTimer
	w := NewWorkers(Options{
		Report: failOnReport(t), CompilationCache: newCompilation(t), Workers: 0, IdleWorkers: 2,
		AfterFunc: func(_ time.Duration, _ func()) Timer {
			tr := &recordingTimer{}
			timers = append(timers, tr)
			return tr
		},
	})
	if cfg := w.getRuntimeConfig(); cfg == nil {
		t.Fatal("expected non-nil RuntimeConfig when compilation cache is set")
	}
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := w.openBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
	if len(timers) != 1 || timers[0].stopped {
		t.Fatalf("expected 1 active timer after closing document, got %d", len(timers))
	}
	doc2, err := w.openBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !timers[0].stopped {
		t.Error("expected first timer to be stopped when acquiring pool again")
	}
	closeAndCheck(t, doc2)
	if len(timers) != 2 || timers[1].stopped {
		t.Fatalf("expected 2nd active timer after closing second document, got %d", len(timers))
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if !timers[1].stopped {
		t.Error("expected second timer to be stopped on Workers.Close()")
	}
}

func TestOpenSourceSizeValidation(t *testing.T) {
	w := newWorkers(t)
	fcNeg := &seekFailCloser{}
	if _, err := w.openSource(source{reader: bytes.NewReader(nil), size: 0, file: fcNeg}); !errors.Is(err, ErrNotPDF) || !fcNeg.isClosed {
		t.Errorf("expected ErrNotPDF and closed file for size 0, got %v, closed=%v", err, fcNeg.isClosed)
	}
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	s := &seekFailCloser{}
	doc, err := w.openReader(&struct {
		*bytes.Reader
		io.Closer
	}{Reader: bytes.NewReader(raw), Closer: s})
	if err != nil {
		t.Fatal(err)
	}
	closeAndCheck(t, doc)
}

func closeAndCheck(tb testing.TB, c io.Closer) {
	tb.Helper()
	if err := c.Close(); err != nil {
		tb.Errorf("close: %v", err)
	}
}
