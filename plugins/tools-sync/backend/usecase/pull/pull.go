// Package pull brings the changes of a synced vault from the server into the vault on this device.
package pull

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// ErrCorruptBlob means a blob the server sent is not the one its revision names, or does not open.
var ErrCorruptBlob = note.ErrCorruptBlob

// State is what a pull keeps of the vault.
type State interface {
	ports.Links
	ports.Records
}

// Deps are what the use case needs.
type Deps struct {
	Revisions ports.Revisions
	Blobs     ports.BlobDownloader
	Crypto    ports.Crypto
	Host      ports.Host
	State     State
	LazyHeavy bool
}

// PullChanges reads the changes after the position of the vault and applies them.
type PullChanges struct{ deps Deps }

// NewPullChanges returns the use case over the dependencies.
func NewPullChanges(deps Deps) *PullChanges { return &PullChanges{deps: deps} }

// Execute applies every change after the position kept in the link. The position stops before the first note
// that is left as it was, so the next pull reads it again.
func (u *PullChanges) Execute(ctx context.Context, in transfer.PullInput) (transfer.PullResult, error) {
	link, err := u.deps.State.ReadLink(ctx, in.VaultID)
	if err != nil {
		return transfer.PullResult{}, err
	}
	records, err := u.deps.State.ReadRecords(ctx, in.VaultID)
	if err != nil {
		return transfer.PullResult{}, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return transfer.PullResult{}, fmt.Errorf("read the remote id of the link: %w", err)
	}
	key, err := u.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return transfer.PullResult{}, fmt.Errorf("open the vault key: %w", err)
	}
	r := &run{
		deps: u.deps, in: in, files: u.deps.Host.Files(in.VaultID), remote: remote, key: [32]byte(key), link: link, records: records,
		result: transfer.PullResult{Applied: nil, Conflicts: nil, Unverified: 0}, holdSeq: 0, isHeld: false, listed: nil,
	}
	err = r.execute(ctx)
	return r.result, err
}

func (r *run) execute(ctx context.Context) error {
	since := r.link.ServerSeq
	for {
		page, err := r.deps.Revisions.GetChanges(ctx, r.remote, since, 0)
		if err != nil {
			return err
		}
		for _, stored := range page.GetChanges() {
			if err := r.apply(ctx, stored); err != nil {
				return err
			}
		}
		if err := r.save(ctx, page.GetNextSeq()); err != nil {
			return err
		}
		if !page.GetHasMore() {
			return nil
		}
		since = page.GetNextSeq()
	}
}

func (r *run) save(ctx context.Context, next uint64) error {
	if err := r.deps.State.WriteRecords(ctx, r.in.VaultID, r.records); err != nil {
		return err
	}
	r.link.ServerSeq = next
	if r.isHeld {
		r.link.ServerSeq = r.holdSeq
	}
	return r.deps.State.WriteLink(ctx, r.in.VaultID, r.link)
}

func (r *run) hold(seq uint64) {
	if !r.isHeld || seq-1 < r.holdSeq {
		r.holdSeq, r.isHeld = seq-1, true
	}
}
