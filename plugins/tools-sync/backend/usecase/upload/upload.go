// Package upload seals one file of any size and sends it to the sync server, in one body or in parts, and goes on
// from the parts the server holds after a break.
package upload

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrFileChanged means the file changed while it was read; the pass tries it again later.
var ErrFileChanged = errors.New("the file changed while it was sent")

// Deps are what the use case needs.
type Deps struct {
	Uploads ports.Uploads
	Entries ports.UploadEntries
	Crypto  ports.Crypto
	Host    ports.Host
	Random  io.Reader
	Digests ports.Digests
}

// UploadFile sends one file.
type UploadFile struct{ deps Deps }

// NewUploadFile returns the use case over the dependencies.
func NewUploadFile(deps Deps) *UploadFile { return &UploadFile{deps: deps} }

// Execute sends the file whole or in parts and returns what its revision names.
func (u *UploadFile) Execute(ctx context.Context, in transfer.UploadInput) (transfer.UploadResult, error) {
	return (&run{deps: u.deps, in: in, plain: nil}).execute(ctx)
}

// Forget drops the kept state of the upload of a file once its revision is committed.
func (u *UploadFile) Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error {
	entries, err := u.deps.Entries.ReadUploads(ctx, vault)
	if err != nil {
		return err
	}
	kept := withoutFile(entries, fileID)
	if len(kept) == len(entries) {
		return nil
	}
	return u.deps.Entries.WriteUploads(ctx, vault, kept)
}

// plainReader reads a file by ranges and ends with ErrFileChanged when its stat is not the one it began with.
type plainReader struct {
	read func(start, length int64) (host.ByteRange, error)
	stat domain.FileStat
	size int64
}

func (r *plainReader) ReadAt(p []byte, off int64) (int, error) {
	count := 0
	for count < len(p) && off+int64(count) < r.size {
		length := min(int64(len(p)-count), host.MaxReadBytes)
		got, err := r.read(off+int64(count), length)
		if err != nil {
			return count, err
		}
		if got.Stat.Hash != r.stat.Hash || got.Stat.Size != r.stat.Size || !got.Stat.ModTime.Equal(r.stat.ModTime) {
			return count, ErrFileChanged
		}
		if len(got.Content) == 0 {
			return count, io.ErrUnexpectedEOF
		}
		count += copy(p[count:], got.Content)
	}
	if count < len(p) {
		return count, io.EOF
	}
	return count, nil
}

func isBig(storedSize int64) bool { return storedSize > note.BigFileLimit }

func isText(path string) bool { return note.Kind(path) != note.KindBinary }

func readAll(r io.ReaderAt, size int64) ([]byte, error) {
	whole := make([]byte, size)
	n, err := r.ReadAt(whole, 0)
	if err != nil && (!errors.Is(err, io.EOF) || int64(n) != size) {
		return nil, err
	}
	return whole, nil
}
