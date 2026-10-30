package pdf

import (
	"bytes"
	"errors"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"io"
	"os"
	"testing"
)

type seekFailCloser struct {
	err             error
	failOnSeekEnd   bool
	failOnSeekStart bool
	isClosed        bool
}

func (s *seekFailCloser) Read([]byte) (int, error) { return 0, io.EOF }

func (s *seekFailCloser) Seek(offset int64, whence int) (int64, error) {
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
			doc, err := openReader(fc)
			if err == nil {
				if doc != nil {
					cleanup.Close(t, doc)
				}
				t.Fatal("openReader succeeded, want seek error")
			}
			if !fc.isClosed {
				t.Error("file was not closed on seek failure")
			}
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("error %v does not wrap %v", err, tc.expectedErr)
			}
			if unwrapped := errors.Unwrap(err); unwrapped != tc.expectedErr {
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
	_, err := openReader(fc)
	if !errors.Is(err, ErrNotPDF) {
		t.Errorf("expected ErrNotPDF for empty file, got %v", err)
	}
	if !fc.isClosed {
		t.Error("expected empty file to be closed")
	}
}

func TestOpenReaderInvalidContentClosesFile(t *testing.T) {
	fc := &trackingCloser{Reader: bytes.NewReader([]byte("not a valid pdf content"))}
	_, err := openReader(fc)
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
	doc, err := openReader(fc)
	if err != nil {
		t.Fatal(err)
	}
	if fc.isClosed {
		t.Error("file should not be closed while document is open")
	}
	cleanup.Close(t, doc)
	if !fc.isClosed {
		t.Error("doc.close() did not close file")
	}
}

func TestDocumentTextInvalidPageIndex(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := openBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, doc)

	if got := doc.text(-1); got != "" {
		t.Errorf("doc.text(-1) = %q, want empty string", got)
	}
	if got := doc.text(9999); got != "" {
		t.Errorf("doc.text(9999) = %q, want empty string", got)
	}
}
