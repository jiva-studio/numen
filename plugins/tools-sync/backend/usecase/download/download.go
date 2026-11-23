// Package download brings the body of one file of any size down from the sync server into the store folder of the
// plugin as a staged file, opening it chunk by chunk, and goes on from the chunk it reached after a break.
package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/blobcipher"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var (
	// ErrIntegrity means the bytes the server sent are not the blob its revision names.
	ErrIntegrity = transfer.ErrIntegrity
	// ErrBlobMissing means the server has no blob for a revision.
	ErrBlobMissing = transfer.ErrBlobMissing
)

const (
	saveEvery = 16
	stageSize = 1 << 20
)

// Blobs reads the blobs on the server.
type Blobs interface {
	// GetBlob returns a whole blob of up to note.BigFileLimit bytes.
	GetBlob(ctx context.Context, vault, blob []byte) ([]byte, error)
	// OpenBlob streams the blob from the byte from; the caller closes it.
	OpenBlob(ctx context.Context, vault, blob []byte, from int64) (io.ReadCloser, error)
}

// Entry is a download that is not finished, kept between runs. Its hash state is the state of the hash of the stored
// bytes, never of the plain ones.
type Entry struct {
	FileID          []byte `json:"fileId"`
	Path            string `json:"path"`
	BlobID          []byte `json:"blobId"`
	Header          []byte `json:"header"`
	NextChunk       uint64 `json:"nextChunk"`
	StoredHashState []byte `json:"storedHashState"`
	StagedName      string `json:"stagedName"`
	MetaHash        string `json:"metaHash"`
	HeadRev         []byte `json:"headRev"`
}

// Entries keeps the unfinished downloads of each vault.
type Entries interface {
	ReadDownloads(ctx context.Context, id domain.VaultID) ([]Entry, error)
	WriteDownloads(ctx context.Context, id domain.VaultID, entries []Entry) error
}

// Deps are what the use case needs.
type Deps struct {
	Blobs   Blobs
	Entries Entries
	Host    ports.Host
}

// DownloadFile stages the body of a file.
type DownloadFile struct{ deps Deps }

// NewDownloadFile returns the use case over the dependencies.
func NewDownloadFile(deps Deps) *DownloadFile { return &DownloadFile{deps: deps} }

// Execute brings the file down into a staged file of the store folder, checks it against the hash of its blob and
// its plain hash, and returns the name of the staged file.
func (u *DownloadFile) Execute(ctx context.Context, in transfer.DownloadInput) (string, error) {
	r := &run{deps: u.deps, in: in, name: note.StoredName(in.FileID), store: u.deps.Host.Store(in.Vault)}
	var err error
	if in.BlobSize <= note.BigFileLimit {
		err = r.stageWhole(ctx)
	} else {
		err = r.stageStream(ctx)
	}
	if errors.Is(err, ErrIntegrity) || errors.Is(err, ErrBlobMissing) {
		return "", errors.Join(err, r.discard(ctx))
	}
	return r.name, err
}

// Forget drops the kept state of the download of a file once it is placed.
func (u *DownloadFile) Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error {
	entries, err := u.deps.Entries.ReadDownloads(ctx, vault)
	if err != nil {
		return err
	}
	return u.deps.Entries.WriteDownloads(ctx, vault, withoutFile(entries, fileID))
}

// Discard deletes the staged file of a download and its state.
func (u *DownloadFile) Discard(ctx context.Context, vault domain.VaultID, fileID []byte) error {
	r := &run{deps: u.deps, in: transfer.DownloadInput{Vault: vault, FileID: fileID}, name: note.StoredName(fileID), store: u.deps.Host.Store(vault)}
	return r.discard(ctx)
}

// Sweep deletes the staged files of the vault that no kept download names.
func (u *DownloadFile) Sweep(ctx context.Context, vault domain.VaultID) error {
	entries, err := u.deps.Entries.ReadDownloads(ctx, vault)
	if err != nil {
		return err
	}
	store := u.deps.Host.Store(vault)
	files, err := store.List(ctx)
	if err != nil {
		return err
	}
	var failure error
	for _, file := range files {
		isStaged := len(file.Name) > len(".part") && file.Name[len(file.Name)-len(".part"):] == ".part"
		if isStaged && !slices.ContainsFunc(entries, func(e Entry) bool { return e.StagedName == file.Name }) {
			failure = errors.Join(failure, store.Delete(ctx, file.Name))
		}
	}
	return failure
}

func withoutFile(entries []Entry, fileID []byte) []Entry {
	return slices.DeleteFunc(slices.Clone(entries), func(e Entry) bool { return bytes.Equal(e.FileID, fileID) })
}

type run struct {
	deps  Deps
	in    transfer.DownloadInput
	name  string
	store ports.Store
}

func (r *run) discard(ctx context.Context) error {
	entries, err := r.deps.Entries.ReadDownloads(ctx, r.in.Vault)
	if err != nil {
		return err
	}
	return errors.Join(r.store.Delete(ctx, r.name), r.deps.Entries.WriteDownloads(ctx, r.in.Vault, withoutFile(entries, r.in.FileID)))
}

// stageWhole downloads a blob of up to note.BigFileLimit bytes in one request and opens it whole.
func (r *run) stageWhole(ctx context.Context) error {
	blob, err := r.deps.Blobs.GetBlob(ctx, r.in.RemoteID, r.in.BlobID)
	if err != nil {
		return err
	}
	if sum := sha256.Sum256(blob); !bytes.Equal(sum[:], r.in.BlobHash) {
		return ErrIntegrity
	}
	plain, err := blobcipher.Open(r.in.Key, [16]byte(r.in.FileID), blob)
	if err != nil || note.Hash(plain) != r.in.Hash {
		return ErrIntegrity
	}
	if err := r.store.Delete(ctx, r.name); err != nil {
		return err
	}
	for start := 0; start == 0 || start < len(plain); start += stageSize {
		if err := r.store.Append(ctx, r.name, plain[start:min(start+stageSize, len(plain))]); err != nil {
			return err
		}
	}
	return nil
}
