// Package pull brings the changes of a synced vault from the server into the vault on this device.
package pull

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrCorruptBlob means a blob the server sent is not the one its revision names, or does not open.
var ErrCorruptBlob = note.ErrCorruptBlob

const (
	// KeyIntegrity is the health item of a file from the server that does not match its revision.
	KeyIntegrity = "integrity-error"
)

// State is what a pull keeps of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Downloader stages the body of a file that is not a small note.
type Downloader interface {
	Execute(ctx context.Context, in transfer.DownloadInput) (string, error)
	Forget(ctx context.Context, vault domain.VaultID, fileID []byte) error
	Discard(ctx context.Context, vault domain.VaultID, fileID []byte) error
	Sweep(ctx context.Context, vault domain.VaultID) error
}

// Guard reports whether a pass may remove files in the vault.
type Guard interface {
	Execute(ctx context.Context, in transfer.GuardInput) (transfer.Verdict, error)
}

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobDownloader
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
	Downloads Downloader
	Guard     Guard
	Conflicts ports.ConflictResolver
	Clock     ports.Clock
	Health    ports.Health
	LazyHeavy bool
	// Retries holds when a file the host refused is tried again. It is optional: a pull without it keeps its own, so a
	// pass built anew for each run forgets the waits.
	Retries *Retries
}

// PullChanges reads the changes after the position of the vault and applies them.
type PullChanges struct {
	deps    Deps
	retries *Retries
}

// NewPullChanges returns the use case over the dependencies.
func NewPullChanges(deps Deps) *PullChanges {
	retries := deps.Retries
	if retries == nil {
		retries = NewRetries()
	}
	return &PullChanges{deps: deps, retries: retries}
}

// Execute applies every change after the position kept in the link, unless the server announced a restore this device
// has not run or went back to an earlier state: then it applies nothing and the result says which. The position stops before the first file
// that is left as it was, so the next pull reads it again.
func (u *PullChanges) Execute(ctx context.Context, in ports.PullInput) (restore.PullResult, error) {
	link, err := u.deps.State.ReadLink(ctx, in.VaultID)
	if err != nil {
		return restore.PullResult{}, err
	}
	records, err := u.deps.State.ReadRecords(ctx, in.VaultID)
	if err != nil {
		return restore.PullResult{}, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return restore.PullResult{}, fmt.Errorf("read the remote id of the link: %w", err)
	}
	keys, err := vaultlink.OpenKeys(ctx, u.deps.Crypto, link)
	if err != nil {
		return restore.PullResult{}, err
	}
	r := &run{
		deps: u.deps, in: in, files: u.deps.Host.GetFiles(in.VaultID), moves: u.deps.Host.GetTransfers(in.VaultID), remote: remote,
		keys: keys, link: link, records: records,
		result:  restore.PullResult{Verdict: restore.Same, EpochID: nil, Applied: nil, Conflicts: nil, Unverified: 0, Resolved: nil},
		holdSeq: 0, isHeld: false, next: 0,
		listed: nil, removals: nil, removalSeq: 0, isListing: false, rolledBack: nil, failures: nil, unsaved: nil, retries: u.retries,
	}
	if err = r.execute(ctx); err == nil && r.result.Verdict == restore.Same {
		err = r.reportRollbacks(ctx)
	}
	return r.result, err
}

func (r *run) execute(ctx context.Context) error {
	if err := r.deps.Downloads.Sweep(ctx, r.in.VaultID); err != nil {
		return err
	}
	since := r.link.ServerSeq
	for pages := 1; ; pages++ {
		page, err := r.deps.Revisions.GetChanges(ctx, r.remote, since, 0)
		if err != nil {
			return err
		}
		r.result.EpochID = page.GetRestoreEpochId()
		r.result.Verdict = restore.GetVerdict(
			restore.Memory{EpochID: r.link.RestoreEpochID, Cursor: r.link.ServerSeq, HighSeq: r.link.HighSeq, PinnedKey: nil},
			restore.Look{EpochID: r.result.EpochID, HeadSeq: page.GetHeadSeq(), SignKey: nil},
		)
		if r.result.Verdict != restore.Same {
			return nil
		}
		if err := r.raiseHighSeq(ctx, page.GetHeadSeq()); err != nil {
			return err
		}
		if page.GetReset_() {
			return r.reconcile(ctx)
		}
		if err := r.takePage(ctx, page); err != nil {
			return err
		}
		if !page.GetHasMore() {
			return r.finish(ctx)
		}
		if err := syncprotocol.CheckPage(pages, since, page.GetNextSeq(), page.GetHasMore()); err != nil {
			return err
		}
		since = page.GetNextSeq()
	}
}

func (r *run) raiseHighSeq(ctx context.Context, head uint64) error {
	if head <= r.link.HighSeq {
		return nil
	}
	r.link.HighSeq = head
	return r.deps.State.UpdateLink(ctx, r.in.VaultID, func(fresh *vaultlink.Link) error {
		fresh.HighSeq = max(fresh.HighSeq, head)
		return nil
	})
}

func (r *run) takePage(ctx context.Context, page *syncv1.GetChangesResponse) error {
	if err := r.applyPage(ctx, page.GetChanges()); err != nil {
		return err
	}
	r.next = page.GetNextSeq()
	return r.save(ctx)
}

func (r *run) finish(ctx context.Context) error {
	if err := r.settleRemovals(ctx); err != nil {
		return err
	}
	if err := r.reportUnsaved(ctx); err != nil {
		return err
	}
	return errors.Join(r.failures...)
}

func (r *run) save(ctx context.Context) error {
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	if r.isListing {
		return nil
	}
	r.link.ServerSeq = r.next
	if r.isHeld {
		r.link.ServerSeq = min(r.next, r.holdSeq)
	}
	if len(r.removals) > 0 {
		r.link.ServerSeq = min(r.link.ServerSeq, r.removalSeq-1)
	}
	return r.deps.State.UpdateLink(ctx, r.in.VaultID, func(fresh *vaultlink.Link) error {
		fresh.ServerSeq = r.link.ServerSeq
		fresh.MergeRetired(r.link.Retired)
		return nil
	})
}

func (r *run) hold(seq uint64) {
	if !r.isHeld || seq-1 < r.holdSeq {
		r.holdSeq, r.isHeld = seq-1, true
	}
}

func isBlobFailure(err error) bool {
	return errors.Is(err, errBlobMissing) || errors.Is(err, ErrCorruptBlob)
}
