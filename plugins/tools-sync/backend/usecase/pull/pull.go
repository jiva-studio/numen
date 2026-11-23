// Package pull brings the changes of a synced vault from the server into the vault on this device.
package pull

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrCorruptBlob means a blob the server sent is not the one its revision names, or does not open.
var ErrCorruptBlob = errors.New("the blob of a revision is damaged")

const (
	// KeyIntegrity is the health item of a file from the server that does not match its revision.
	KeyIntegrity = "integrity-error"
	// KeyNeedsFullCheck is the health item of a server that asks for the vault to be compared again.
	KeyNeedsFullCheck = "needs-full-check"
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
	Health    ports.Health
}

// PullChanges reads the changes after the position of the vault and applies them.
type PullChanges struct{ deps Deps }

// NewPullChanges returns the use case over the dependencies.
func NewPullChanges(deps Deps) *PullChanges { return &PullChanges{deps: deps} }

// Execute applies every change after the position kept in the link, unless the server announced a restore this device
// has not run or went back to an earlier state: then it applies nothing and the result says which. The position stops before the first file
// that is left as it was, so the next pull reads it again.
func (u *PullChanges) Execute(ctx context.Context, in transfer.PullInput) (restore.PullResult, error) {
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
	key, err := u.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return restore.PullResult{}, fmt.Errorf("open the vault key: %w", err)
	}
	r := &run{
		deps: u.deps, in: in, files: u.deps.Host.Files(in.VaultID), moves: u.deps.Host.Transfers(in.VaultID), remote: remote,
		key: [32]byte(key), link: link, records: records,
		result:  restore.PullResult{Verdict: restore.Same, EpochID: nil, Applied: nil, Conflicts: nil, Unverified: 0, IsReset: false},
		holdSeq: 0, isHeld: false, next: 0,
		removals: nil, removalSeq: 0, hasRemovals: false,
	}
	err = r.execute(ctx)
	return r.result, err
}

func (r *run) execute(ctx context.Context) error {
	if err := r.deps.Downloads.Sweep(ctx, r.in.VaultID); err != nil {
		return err
	}
	since := r.link.ServerSeq
	for {
		page, err := r.deps.Revisions.GetChanges(ctx, r.remote, since, 0)
		if err != nil {
			return err
		}
		r.result.EpochID = page.GetRestoreEpochId()
		r.result.Verdict = restore.GetVerdict(
			restore.Memory{EpochID: r.link.RestoreEpochID, Cursor: r.link.ServerSeq, PinnedKey: nil},
			restore.Look{EpochID: r.result.EpochID, HeadSeq: page.GetHeadSeq(), SignKey: nil},
		)
		if r.result.Verdict != restore.Same {
			return nil
		}
		if page.GetReset_() {
			r.result.IsReset = true
			return r.deps.Health.Report(ctx, health.Item{
				Key: KeyNeedsFullCheck, Severity: health.Warning, Text: "Sync needs to compare the vault again", Detail: "", Action: nil,
			})
		}
		if err := r.applyPage(ctx, page.GetChanges()); err != nil {
			return err
		}
		r.next = page.GetNextSeq()
		if err := r.save(ctx); err != nil {
			return err
		}
		if !page.GetHasMore() {
			return r.settleRemovals(ctx)
		}
		since = page.GetNextSeq()
	}
}

func (r *run) save(ctx context.Context) error {
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	r.link.ServerSeq = r.next
	if r.isHeld {
		r.link.ServerSeq = min(r.next, r.holdSeq)
	}
	if r.hasRemovals {
		r.link.ServerSeq = min(r.link.ServerSeq, r.removalSeq-1)
	}
	return r.deps.State.WriteLink(ctx, r.in.VaultID, r.link)
}

func (r *run) hold(seq uint64) {
	if !r.isHeld || seq-1 < r.holdSeq {
		r.holdSeq, r.isHeld = seq-1, true
	}
}
