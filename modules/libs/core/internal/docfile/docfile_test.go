package docfile_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/docfile"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

type plainReader struct {
	port.VaultReader
	opened int
}

func (r *plainReader) Open(context.Context, string) (io.ReadSeekCloser, error) {
	r.opened++
	return nopFile{bytes.NewReader(nil)}, nil
}

type sharingReader struct {
	plainReader
	shared int
}

func (r *sharingReader) OpenShared(context.Context, string) (io.ReadSeekCloser, error) {
	r.shared++
	return nopFile{bytes.NewReader(nil)}, nil
}

type nopFile struct{ *bytes.Reader }

func (nopFile) Close() error { return nil }

func TestADocumentFileIsOpenedSharedWhereTheReaderCanShare(t *testing.T) {
	sharing := &sharingReader{}
	if _, err := docfile.Open(t.Context(), sharing, "a.pdf"); err != nil {
		t.Fatal(err)
	}
	if sharing.shared != 1 || sharing.opened != 0 {
		t.Errorf("opened shared %d times and plain %d times, want 1 and 0", sharing.shared, sharing.opened)
	}
}

func TestADocumentFileIsOpenedPlainWhereTheReaderCannotShare(t *testing.T) {
	plain := &plainReader{}
	if _, err := docfile.Open(t.Context(), plain, "a.pdf"); err != nil {
		t.Fatal(err)
	}
	if plain.opened != 1 {
		t.Errorf("opened plain %d times, want 1", plain.opened)
	}
}
