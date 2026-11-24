// Package fetchfile downloads one absent file of a synced vault and places it in the vault.
package fetchfile

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// RangeSize is how many bytes one request of a download asks for.
const RangeSize = 1 << 20

var (
	// ErrBlobMissing means the server answered and holds no blob for the file.
	ErrBlobMissing = errors.New("the server has no blob for the file")
	// ErrNotAbsent means the vault does not list the file as absent.
	ErrNotAbsent = errors.New("the file is not listed as absent")
	// ErrHeadChanged means the head of the file on the server is not the one the vault lists.
	ErrHeadChanged = errors.New("the head of the file is not the one listed")
)

// State is the link and the records of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobDownloader
	Ranges    ports.BlobRanges
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
}

// FetchFile downloads the blob of an absent file in ranges, keeps what came in the store folder of the plugin
// so that a download resumes, and fills the file once its hash is the one of the entry.
type FetchFile struct {
	deps    Deps
	mu      sync.Mutex
	running map[string]*download
}

type download struct {
	done chan struct{}
	err  error
}

// NewFetchFile returns the use case over the dependencies.
func NewFetchFile(deps Deps) *FetchFile {
	return &FetchFile{deps: deps, mu: sync.Mutex{}, running: map[string]*download{}}
}

// Execute downloads the file at path. Two calls for one path run one download, and the other waits for it.
func (u *FetchFile) Execute(ctx context.Context, id domain.VaultID, path string) error {
	key := string(id) + "\x00" + path
	u.mu.Lock()
	if running, isRunning := u.running[key]; isRunning {
		u.mu.Unlock()
		select {
		case <-running.done:
			return running.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	own := &download{done: make(chan struct{}), err: nil}
	u.running[key] = own
	u.mu.Unlock()
	own.err = u.run(ctx, id, path)
	u.mu.Lock()
	delete(u.running, key)
	u.mu.Unlock()
	close(own.done)
	return own.err
}

type target struct {
	id     domain.VaultID
	path   string
	remote []byte
	record note.FileRecord
	entry  host.AbsentFile
}

func (u *FetchFile) run(ctx context.Context, id domain.VaultID, path string) error {
	t, key, err := u.open(ctx, id, path)
	if err != nil {
		return err
	}
	rev, meta, err := u.readHead(ctx, t, key)
	if err != nil {
		return err
	}
	blob, err := u.readBlob(ctx, t, rev)
	if err != nil {
		return err
	}
	plain, err := note.OpenBlob(meta, rev.GetFileId(), rev.GetBlobHash(), blob)
	if err != nil {
		return u.dropPart(ctx, t, rev, err)
	}
	store := u.deps.Host.Store(id)
	name := "fetch-" + hex.EncodeToString(t.record.FileID) + ".bin"
	if err := store.Write(ctx, name, plain); err != nil {
		return err
	}
	fillErr := u.deps.Host.Absent(id).FillFile(ctx, path, name)
	return errors.Join(fillErr, store.Write(ctx, name, nil), u.dropPart(ctx, t, rev, nil))
}

func (u *FetchFile) open(ctx context.Context, id domain.VaultID, path string) (target, [32]byte, error) {
	listed, err := u.deps.Host.Absent(id).ListAbsentFiles(ctx)
	if err != nil {
		return target{}, [32]byte{}, err
	}
	at := slices.IndexFunc(listed, func(one host.AbsentFile) bool { return one.Path == path })
	if at < 0 {
		return target{}, [32]byte{}, fmt.Errorf("download %s: %w", path, ErrNotAbsent)
	}
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return target{}, [32]byte{}, err
	}
	recordAt := slices.IndexFunc(records, func(one note.FileRecord) bool { return one.Path == path })
	if recordAt < 0 {
		return target{}, [32]byte{}, fmt.Errorf("download %s: no record of the file: %w", path, ErrNotAbsent)
	}
	link, err := u.deps.State.ReadLink(ctx, id)
	if err != nil {
		return target{}, [32]byte{}, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return target{}, [32]byte{}, fmt.Errorf("read the remote id of the link: %w", err)
	}
	key, err := u.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return target{}, [32]byte{}, fmt.Errorf("open the vault key: %w", err)
	}
	return target{id: id, path: path, remote: remote, record: records[recordAt], entry: listed[at]}, [32]byte(key), nil
}

func (u *FetchFile) readHead(ctx context.Context, t target, key [32]byte) (*syncv1.Revision, note.Meta, error) {
	var since uint64
	for {
		page, err := u.deps.Revisions.GetChanges(ctx, t.remote, since, 0)
		if err != nil {
			return nil, note.Meta{}, err
		}
		for _, stored := range page.GetChanges() {
			rev := stored.GetRevision()
			if !bytes.Equal(rev.GetFileId(), t.record.FileID) || rev.GetDeleted() {
				continue
			}
			meta, err := note.OpenMeta(key, rev.GetVaultId(), rev.GetFileId(), rev.GetEncMeta())
			if err != nil || len(meta.Key) != 32 || meta.Hash != t.entry.Hash {
				return nil, note.Meta{}, fmt.Errorf("download %s: %w", t.path, ErrHeadChanged)
			}
			return rev, meta, nil
		}
		if !page.GetHasMore() {
			return nil, note.Meta{}, fmt.Errorf("download %s: %w", t.path, ErrHeadChanged)
		}
		since = page.GetNextSeq()
	}
}

func getPartName(rev *syncv1.Revision) string {
	return "fetch-" + hex.EncodeToString(rev.GetBlobId()) + ".part"
}

func (u *FetchFile) readBlob(ctx context.Context, t target, rev *syncv1.Revision) ([]byte, error) {
	size := int64(rev.GetBlobSize())
	if size <= 0 || size > syncprotocol.MaxBlobSize {
		return nil, fmt.Errorf("download %s: the blob has a size of %d bytes: %w", t.path, size, note.ErrCorruptBlob)
	}
	downloads, err := u.deps.Blobs.GetBlobUrls(ctx, t.remote, [][]byte{rev.GetBlobId()})
	if err != nil {
		return nil, err
	}
	if len(downloads) != 1 || !downloads[0].GetFound() {
		return nil, ErrBlobMissing
	}
	store := u.deps.Host.Store(t.id)
	name := getPartName(rev)
	part, err := store.Read(ctx, name)
	if err != nil && !errors.Is(err, host.ErrNotExist) {
		return nil, err
	}
	if int64(len(part)) > size {
		part = nil
	}
	for int64(len(part)) < size {
		next, err := u.deps.Ranges.GetBlobRange(ctx, downloads[0].GetUrl(), int64(len(part)), min(RangeSize, size-int64(len(part))))
		if err != nil {
			return nil, err
		}
		if len(next) == 0 {
			return nil, fmt.Errorf("download %s: the server ended the blob at %d of %d bytes: %w", t.path, len(part), size, note.ErrCorruptBlob)
		}
		part = append(part, next...)
		if err := store.Write(ctx, name, part); err != nil {
			return nil, err
		}
	}
	return part, nil
}

func (u *FetchFile) dropPart(ctx context.Context, t target, rev *syncv1.Revision, cause error) error {
	return errors.Join(cause, u.deps.Host.Store(t.id).Write(ctx, getPartName(rev), nil))
}
