// Package fetchfile downloads one absent file of a synced vault and places it in the vault.
package fetchfile

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// RangeSize is how many bytes one request of a download asks for.
const RangeSize = 1 << 20

var (
	// ErrBlobMissing means the server answered and holds no blob for the file.
	ErrBlobMissing = transfer.ErrBlobMissing
	// ErrNotAbsent means the vault does not list the file as absent.
	ErrNotAbsent = errors.New("the file is not listed as absent")
	// ErrHeadChanged means the head of the file on the server is not the one the vault lists.
	ErrHeadChanged = errors.New("the head of the file is not the one listed")

	errHeld = errors.New("the file is already placed")
)

// State is the link and the records of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Stager brings a file of more than the whole-file limit down chunk by chunk into a staged file of the store
// folder, and goes on from the chunk it reached after a break.
type Stager interface {
	Execute(ctx context.Context, in transfer.DownloadInput) (string, error)
	Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error
}

// Deps are what the use case needs.
type Deps struct {
	Stage     Stager
	Revisions ports.Revisions
	Blobs     ports.BlobReader
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
	record transfer.FileRecord
	entry  host.AbsentFile
}

func (u *FetchFile) run(ctx context.Context, id domain.VaultID, path string) error {
	t, key, err := u.open(ctx, id, path)
	if errors.Is(err, errHeld) {
		return nil
	}
	if err != nil {
		return err
	}
	rev, meta, err := u.readHead(ctx, t, key)
	if err != nil {
		return err
	}
	if rev.GetBlobSize() > note.BigFileLimit {
		return u.runLarge(ctx, t, rev, meta)
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
	fillErr := u.deps.Host.GetAbsent(id).FillFile(ctx, path, name)
	return errors.Join(fillErr, store.Write(ctx, name, nil), u.dropPart(ctx, t, rev, nil))
}

func (u *FetchFile) runLarge(ctx context.Context, t target, rev *syncv1.Revision, meta note.Meta) error {
	in := transfer.DownloadInput{
		Vault: t.id, RemoteID: t.remote, FileID: t.record.FileID, Path: t.path, BlobID: rev.GetBlobId(), BlobHash: rev.GetBlobHash(),
		BlobSize: rev.GetBlobSize(), Key: [32]byte(meta.Key), Hash: meta.Hash, Size: meta.Size, HeadRev: nil, Baseline: nil,
	}
	name, err := u.deps.Stage.Execute(ctx, in)
	if errors.Is(err, transfer.ErrIntegrity) {
		return errors.Join(note.ErrCorruptBlob, err)
	}
	if err != nil {
		return err
	}
	fillErr := u.deps.Host.GetAbsent(t.id).FillFile(ctx, t.path, name)
	return errors.Join(fillErr, u.deps.Host.Store(t.id).Delete(ctx, name), u.deps.Stage.Forget(ctx, t.id, t.record.FileID))
}

func (u *FetchFile) open(ctx context.Context, id domain.VaultID, path string) (target, [32]byte, error) {
	listed, err := u.deps.Host.GetAbsent(id).ListAbsentFiles(ctx)
	if err != nil {
		return target{}, [32]byte{}, err
	}
	at := slices.IndexFunc(listed, func(one host.AbsentFile) bool { return one.Path == path })
	if at < 0 {
		return target{}, [32]byte{}, u.checkHeld(ctx, id, path)
	}
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return target{}, [32]byte{}, err
	}
	recordAt := slices.IndexFunc(records, func(one transfer.FileRecord) bool { return one.Path == path })
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
	for pages := 1; ; pages++ {
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
		if err := syncprotocol.CheckPage(pages, since, page.GetNextSeq(), page.GetHasMore()); err != nil {
			return nil, note.Meta{}, err
		}
		since = page.GetNextSeq()
	}
}

func getPartName(rev *syncv1.Revision) string {
	return "fetch-" + hex.EncodeToString(rev.GetBlobId()) + ".part"
}

func (u *FetchFile) readBlob(ctx context.Context, t target, rev *syncv1.Revision) ([]byte, error) {
	size := int64(rev.GetBlobSize())
	if size <= 0 {
		return nil, fmt.Errorf("download %s: the blob has a size of %d bytes: %w", t.path, size, note.ErrCorruptBlob)
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
		next, err := u.readRange(ctx, t, rev, int64(len(part)), min(RangeSize, size-int64(len(part))))
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

func (u *FetchFile) readRange(ctx context.Context, t target, rev *syncv1.Revision, from, length int64) ([]byte, error) {
	body, err := u.deps.Blobs.OpenBlob(ctx, t.remote, rev.GetBlobId(), from)
	if err != nil {
		return nil, err
	}
	next, readErr := io.ReadAll(io.LimitReader(body, length))
	return next, errors.Join(readErr, body.Close())
}

func (u *FetchFile) checkHeld(ctx context.Context, id domain.VaultID, path string) error {
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(records, func(one transfer.FileRecord) bool { return one.Path == path }) {
		return fmt.Errorf("download %s: %w", path, ErrNotAbsent)
	}
	if _, err := u.deps.Host.GetFiles(id).StatFile(ctx, path); err != nil {
		return fmt.Errorf("download %s: %w", path, ErrNotAbsent)
	}
	return errHeld
}
