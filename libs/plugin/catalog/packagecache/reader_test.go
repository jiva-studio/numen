package packagecache

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
)

type singleChunkEOFReader struct {
	data string
}

func (r singleChunkEOFReader) Read(p []byte) (int, error) {
	return copy(p, r.data), io.EOF
}

func TestCheckedReaderRead(t *testing.T) {
	canceled := &checkedReader{
		getContextError: func() error { return context.Canceled },
		content:         singleChunkEOFReader{data: "abc"},
		hash:            sha256.New(),
		checksum:        "00",
	}
	buf := make([]byte, 8)
	if n, err := canceled.Read(buf); n != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Read = %d, %v; want 0, context.Canceled", n, err)
	}

	mismatch := &checkedReader{
		getContextError: func() error { return nil },
		content:         singleChunkEOFReader{data: "abc"},
		hash:            sha256.New(),
		checksum:        "00",
	}
	if n, err := mismatch.Read(buf); n != 3 || !errors.Is(err, catalog.ErrChecksumMismatch) {
		t.Fatalf("mismatch Read = %d, %v; want 3, ErrChecksumMismatch", n, err)
	}
}
