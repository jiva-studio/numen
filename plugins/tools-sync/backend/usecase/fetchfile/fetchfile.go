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
	"sync/atomic"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
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
	// Hold keeps the staged file of the name in the vault from being swept until the returned function is called.
	Hold(vault domain.VaultID, name string) func()
}

// SharesReader reads the shares of a vault with the keys of their epochs.
type SharesReader interface {
	Execute(ctx context.Context, id domain.VaultID) ([]share.Root, error)
}

// Deps are what the use case needs.
type Deps struct {
	Stage     Stager
	Revisions ports.Revisions
	Blobs     ports.BlobReader
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
	Shares    SharesReader
	Identity  ports.Identity
	Chain     ports.Chain
}

// FetchFile downloads the blob of an absent file in ranges, keeps what came in the store folder of the plugin
// so that a download resumes, and fills the file once its hash is the one of the entry.
type FetchFile struct {
	deps    Deps
	mu      sync.Mutex
	running map[string]*download
}

type download struct {
	done    chan struct{}
	err     error
	waiters int
	cancel  context.CancelFunc
	vault   domain.VaultID
	path    string
	bytes   atomic.Int64
	total   atomic.Int64
	staged  atomic.Bool
}

// Progress is how far a running download has come. A staged download is a large file that the stager brings
// down chunk by chunk, so its bytes are in the entries of the vault and not in Bytes.
type Progress struct {
	Vault    domain.VaultID
	Bytes    int64
	Total    int64
	IsStaged bool
}

// GetProgress returns how far the running download of the file at path has come, and false when none runs.
func (u *FetchFile) GetProgress(path string) (Progress, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for _, own := range u.running {
		if own.path == path {
			total := own.total.Load()
			return Progress{Vault: own.vault, Bytes: min(own.bytes.Load(), total), Total: total, IsStaged: own.staged.Load()}, true
		}
	}
	return Progress{Vault: "", Bytes: 0, Total: 0, IsStaged: false}, false
}

// NewFetchFile returns the use case over the dependencies.
func NewFetchFile(deps Deps) *FetchFile {
	return &FetchFile{deps: deps, mu: sync.Mutex{}, running: map[string]*download{}}
}

// Execute downloads the file at path. Two calls for one path share one download, and each returns its result. The
// download does not end when the call that began it ends; it ends when no call is left waiting for it, and a call
// that comes then waits for that end and starts a download of its own.
func (u *FetchFile) Execute(ctx context.Context, id domain.VaultID, path string) error {
	key := string(id) + "\x00" + path
	for {
		own, isJoined := u.join(ctx, key, id, path)
		if !isJoined {
			select {
			case <-own.done:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		select {
		case <-own.done:
			return own.err
		case <-ctx.Done():
			u.leave(own)
			return ctx.Err()
		}
	}
}

func (u *FetchFile) join(ctx context.Context, key string, id domain.VaultID, path string) (*download, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	own, isRunning := u.running[key]
	if isRunning && own.waiters == 0 {
		return own, false
	}
	if !isRunning {
		runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		own = &download{done: make(chan struct{}), err: nil, waiters: 0, cancel: cancel, vault: id, path: path, bytes: atomic.Int64{}, total: atomic.Int64{}, staged: atomic.Bool{}}
		u.running[key] = own
		go u.finish(runCtx, key, own, id, path)
	}
	own.waiters++
	return own, true
}

func (u *FetchFile) finish(ctx context.Context, key string, own *download, id domain.VaultID, path string) {
	own.err = u.run(ctx, own, id, path)
	u.mu.Lock()
	delete(u.running, key)
	u.mu.Unlock()
	own.cancel()
	close(own.done)
}

func (u *FetchFile) leave(own *download) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if own.waiters--; own.waiters == 0 {
		own.cancel()
	}
}

type target struct {
	id     domain.VaultID
	path   string
	remote []byte
	record transfer.FileRecord
	entry  host.AbsentFile
	own    *download
}

func (u *FetchFile) run(ctx context.Context, own *download, id domain.VaultID, path string) error {
	t, keys, err := u.open(ctx, id, path)
	if errors.Is(err, errHeld) {
		return nil
	}
	if err != nil {
		return err
	}
	t.own = own
	own.total.Store(t.entry.Size)
	rev, meta, err := u.readHead(ctx, t, keys)
	if err != nil {
		return err
	}
	if rev.GetBlobSize() > note.BigFileLimit {
		defer u.deps.Stage.Hold(id, note.GetStoredName(t.record.FileID))()
		return u.runLarge(ctx, t, rev, meta)
	}
	defer u.deps.Stage.Hold(id, getPartName(rev))()
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
	return errors.Join(fillErr, store.Delete(ctx, name), u.dropPart(ctx, t, rev, nil))
}

func (u *FetchFile) runLarge(ctx context.Context, t target, rev *syncv1.Revision, meta note.Meta) error {
	t.own.staged.Store(true)
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

func (u *FetchFile) open(ctx context.Context, id domain.VaultID, path string) (target, []vaultlink.VaultKey, error) {
	listed, err := u.deps.Host.GetAbsent(id).ListAbsentFiles(ctx)
	if err != nil {
		return target{}, nil, err
	}
	at := slices.IndexFunc(listed, func(one host.AbsentFile) bool { return one.Path == path })
	if at < 0 {
		return target{}, nil, u.checkHeld(ctx, id, path)
	}
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return target{}, nil, err
	}
	recordAt := slices.IndexFunc(records, func(one transfer.FileRecord) bool { return one.Path == path })
	if recordAt < 0 {
		return target{}, nil, fmt.Errorf("download %s: no record of the file: %w", path, ErrNotAbsent)
	}
	link, err := u.deps.State.ReadLink(ctx, id)
	if err != nil {
		return target{}, nil, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return target{}, nil, fmt.Errorf("read the remote id of the link: %w", err)
	}
	keys, err := vaultlink.OpenKeys(ctx, u.deps.Crypto, link)
	if err != nil {
		return target{}, nil, err
	}
	return target{id: id, path: path, remote: remote, record: records[recordAt], entry: listed[at], own: nil}, keys, nil
}

func (u *FetchFile) readHead(ctx context.Context, t target, keys []vaultlink.VaultKey) (*syncv1.Revision, note.Meta, error) {
	authors, err := u.getAuthors(ctx)
	if err != nil {
		return nil, note.Meta{}, err
	}
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
			if err := note.CheckRevision(stored, t.remote, authors.IsTrusted); err != nil {
				return nil, note.Meta{}, fmt.Errorf("download %s: %w", t.path, err)
			}
			meta, isOpen, err := u.openMeta(ctx, t.id, keys, rev)
			if err != nil {
				return nil, note.Meta{}, err
			}
			if !isOpen || len(meta.Key) != 32 || meta.Hash != t.entry.Hash {
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

func (u *FetchFile) getAuthors(ctx context.Context) (trust.Authors, error) {
	own, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	stored, err := u.deps.Chain.GetAccountChain(ctx)
	if err != nil {
		return trust.Authors{}, err
	}
	return trust.NewAuthors(own.Chain, stored)
}

func (u *FetchFile) openMeta(ctx context.Context, id domain.VaultID, keys []vaultlink.VaultKey, rev *syncv1.Revision) (note.Meta, bool, error) {
	if len(rev.GetShareRootId()) > 0 {
		if u.deps.Shares == nil {
			return note.Meta{}, false, nil
		}
		roots, err := u.deps.Shares.Execute(ctx, id)
		if err != nil {
			return note.Meta{}, false, err
		}
		root, isFound := share.FindRootByID(roots, rev.GetShareRootId())
		if !isFound {
			return note.Meta{}, false, nil
		}
		key, isKnown := root.Keys.Get(rev.GetEpoch())
		if !isKnown {
			return note.Meta{}, false, nil
		}
		meta, err := share.OpenMeta[note.Meta](key, root.ID, rev.GetFileId(), rev.GetEpoch(), rev.GetEncMeta())
		return meta, err == nil, nil
	}
	at := slices.IndexFunc(keys, func(one vaultlink.VaultKey) bool { return one.Epoch == rev.GetEpoch() && len(one.Key) == 32 })
	if at < 0 {
		return note.Meta{}, false, nil
	}
	meta, err := note.OpenMeta([32]byte(keys[at].Key), rev.GetVaultId(), rev.GetFileId(), rev.GetEncMeta())
	return meta, err == nil, nil
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
		if err := store.Delete(ctx, name); err != nil {
			return nil, err
		}
	}
	t.own.bytes.Store(int64(len(part)))
	for int64(len(part)) < size {
		next, err := u.readRange(ctx, t, rev, int64(len(part)), min(RangeSize, size-int64(len(part))))
		if err != nil {
			return nil, err
		}
		if len(next) == 0 {
			return nil, fmt.Errorf("download %s: the server ended the blob at %d of %d bytes: %w", t.path, len(part), size, note.ErrCorruptBlob)
		}
		part = append(part, next...)
		if err := store.Append(ctx, name, next); err != nil {
			return nil, err
		}
		t.own.bytes.Store(int64(len(part)))
	}
	return part, nil
}

func (u *FetchFile) dropPart(ctx context.Context, t target, rev *syncv1.Revision, cause error) error {
	return errors.Join(cause, u.deps.Host.Store(t.id).Delete(ctx, getPartName(rev)))
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
