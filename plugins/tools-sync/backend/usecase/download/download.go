// Package download brings the body of one file of any size down from the sync server into the store folder of the
// plugin as a staged file, opening it chunk by chunk, and goes on from the chunk it reached after a break.
package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strings"

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

// Deps are what the use case needs.
type Deps struct {
	Blobs   ports.BlobReader
	Entries ports.DownloadEntries
	Host    ports.Host
	Digests ports.Digests
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
	kept := withoutFile(entries, fileID)
	if len(kept) == len(entries) {
		return nil
	}
	return u.deps.Entries.WriteDownloads(ctx, vault, kept)
}

// Discard deletes the staged file of a download and its state.
func (u *DownloadFile) Discard(ctx context.Context, vault domain.VaultID, fileID []byte) error {
	r := &run{deps: u.deps, in: transfer.DownloadInput{
		Vault: vault, RemoteID: nil, FileID: fileID, Path: "", BlobID: nil, BlobHash: nil, BlobSize: 0, Key: [32]byte{}, Hash: "", Size: 0, HeadRev: nil, Baseline: nil,
	}, name: note.StoredName(fileID), store: u.deps.Host.Store(vault)}
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
		isStaged := strings.HasSuffix(file.Name, ".part")
		if isStaged && !slices.ContainsFunc(entries, func(e transfer.DownloadEntry) bool { return e.StagedName == file.Name }) {
			failure = errors.Join(failure, store.Delete(ctx, file.Name))
		}
	}
	return failure
}

func withoutFile(entries []transfer.DownloadEntry, fileID []byte) []transfer.DownloadEntry {
	return slices.DeleteFunc(slices.Clone(entries), func(e transfer.DownloadEntry) bool { return bytes.Equal(e.FileID, fileID) })
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
