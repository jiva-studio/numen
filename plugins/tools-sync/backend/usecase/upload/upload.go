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

var (
	// ErrFileChanged means the file changed while it was read; the pass tries it again later.
	ErrFileChanged = errors.New("the file changed while it was sent")
	// ErrUploadGone means the server has no such reservation any more.
	ErrUploadGone = errors.New("the upload does not exist on the sync server")
)

// Upload is one reservation on the server: its blob, the size of its parts and the parts the server holds.
type Upload struct {
	Vault     []byte
	URL       string
	Blob      []byte
	PartSize  int64
	PartCount int
	Done      []int
}

// Uploads is one way to upload: a body that fits one request is one part of one.
type Uploads interface {
	Start(ctx context.Context, vault []byte, storedSize int64) (Upload, error)
	Resume(ctx context.Context, vault, blob []byte) (Upload, error)
	PutPart(ctx context.Context, up Upload, number int, body []byte) error
	Finish(ctx context.Context, up Upload) error
	Abort(ctx context.Context, up Upload) error
}

// Entry is an upload of a big file that is not finished, kept between runs. Its key is sealed and its hash state is
// the state of the hash of ciphertext only.
type Entry struct {
	FileID          []byte `json:"fileId"`
	Path            string `json:"path"`
	PlainSize       int64  `json:"plainSize"`
	PlainHash       string `json:"plainHash"`
	MTime           int64  `json:"mtime"`
	ParentRev       []byte `json:"parentRev"`
	BlobID          []byte `json:"blobId"`
	PartSize        int64  `json:"partSize"`
	PartCount       int    `json:"partCount"`
	StoredSize      int64  `json:"storedSize"`
	Salt            []byte `json:"salt"`
	SealedKey       []byte `json:"sealedKey"`
	PartsDone       int    `json:"partsDone"`
	StoredHashState []byte `json:"storedHashState"`
}

// Entries keeps the unfinished uploads of each vault.
type Entries interface {
	ReadUploads(ctx context.Context, id domain.VaultID) ([]Entry, error)
	WriteUploads(ctx context.Context, id domain.VaultID, entries []Entry) error
}

// Crypto seals and opens the key of an upload with the keys of the account.
type Crypto interface {
	SealData(ctx context.Context, dataContext, plain []byte) ([]byte, error)
	OpenData(ctx context.Context, dataContext, sealed []byte) ([]byte, error)
}

// Deps are what the use case needs.
type Deps struct {
	Uploads Uploads
	Entries Entries
	Crypto  Crypto
	Host    ports.Host
	Random  io.Reader
}

// UploadFile sends one file.
type UploadFile struct{ deps Deps }

func NewUploadFile(deps Deps) *UploadFile { return &UploadFile{deps: deps} }

// Execute sends the file whole or in parts and returns what its revision names.
func (u *UploadFile) Execute(ctx context.Context, in transfer.UploadInput) (transfer.UploadResult, error) {
	return (&run{deps: u.deps, in: in}).execute(ctx)
}

// Forget drops the kept state of the upload of a file once its revision is committed.
func (u *UploadFile) Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error {
	entries, err := u.deps.Entries.ReadUploads(ctx, vault)
	if err != nil {
		return err
	}
	return u.deps.Entries.WriteUploads(ctx, vault, withoutFile(entries, fileID))
}

// plainReader reads a file by ranges and ends with ErrFileChanged when its stat is not the one it began with.
type plainReader struct {
	ctx    context.Context
	source rangeReader
	path   string
	stat   domain.FileStat
	size   int64
}

func (r *plainReader) ReadAt(p []byte, off int64) (int, error) {
	count := 0
	for count < len(p) && off+int64(count) < r.size {
		length := min(int64(len(p)-count), host.MaxReadBytes)
		got, err := r.source.ReadBytes(r.ctx, r.path, off+int64(count), length)
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
	if err != nil && !(errors.Is(err, io.EOF) && int64(n) == size) {
		return nil, err
	}
	return whole, nil
}

// rangeReader reads bytes of any file as they are.
type rangeReader interface {
	ReadBytes(ctx context.Context, path string, start, length int64) (host.ByteRange, error)
}
