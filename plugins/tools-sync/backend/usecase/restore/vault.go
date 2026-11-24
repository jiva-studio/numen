package restore

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	domainrestore "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const batchSize = 100

// RecoverVault sends the notes of one vault the server lost or holds in an older state, and writes the epoch id to
// the link only after the last note was examined.
type RecoverVault struct{ deps Deps }

// NewRecoverVault returns the use case over the dependencies.
func NewRecoverVault(deps Deps) *RecoverVault { return &RecoverVault{deps: deps} }

// recovery is one run over one vault.
type recovery struct {
	deps    Deps
	id      domain.VaultID
	vault   transfer.Vault
	link    vaultlink.Link
	records []transfer.FileRecord
	hashes  map[string]string
}

// Execute runs the recovery of the vault, continuing what an earlier run left in the link. The content of this
// device is the truth: a note of the device is never replaced by the one of the server.
func (u *RecoverVault) Execute(ctx context.Context, id domain.VaultID) error {
	link, err := u.deps.State.ReadLink(ctx, id)
	if err != nil {
		return err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return fmt.Errorf("read the remote id of the link: %w", err)
	}
	keys, err := vaultlink.OpenKeys(ctx, u.deps.Crypto, link)
	if err != nil {
		return err
	}
	vault := transfer.Vault{ID: remote, Key: [32]byte(keys[0].Key), Later: nil}
	for _, key := range keys[1:] {
		vault.Later = append(vault.Later, transfer.EpochKey{Epoch: key.Epoch, Key: [32]byte(key.Key)})
	}
	records, err := u.deps.State.ReadRecords(ctx, id)
	if err != nil {
		return err
	}
	found, err := u.readHeads(ctx, vault)
	if err != nil {
		return err
	}
	if err := u.markMissingBlobs(ctx, remote, found, records); err != nil {
		return err
	}
	run := &recovery{deps: u.deps, id: id, vault: vault, link: link, records: records, hashes: nil}
	if run.hashes, err = run.readHashes(ctx, found); err != nil {
		return err
	}
	work := run.getWork(domainrestore.PlanRecovery(run.getKnownFiles(), found.byFile, run.hashes))
	if err := run.execute(ctx, work); err != nil {
		return err
	}
	run.link.RestoreEpochID, run.link.Recovery, run.link.ServerSeq = found.epochID, nil, 0
	return u.deps.State.WriteLink(ctx, id, run.link)
}

func (r *recovery) readHashes(ctx context.Context, found heads) (map[string]string, error) {
	paths := map[string]bool{}
	for _, record := range r.records {
		paths[record.Path] = true
	}
	for _, head := range found.byFile {
		if head.Path != "" {
			paths[head.Path] = true
		}
	}
	hashes := make(map[string]string, len(paths))
	files := r.deps.Host.GetFiles(r.id)
	for path := range paths {
		stat, err := files.StatFile(ctx, path)
		if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
			continue
		}
		if err != nil {
			return nil, err
		}
		hashes[path] = stat.Hash
	}
	return hashes, nil
}

func (r *recovery) getWork(plan []domainrestore.Action) []domainrestore.Action {
	var work []domainrestore.Action
	for _, action := range plan {
		isWork := action.Kind == domainrestore.Send || action.Kind == domainrestore.Recreate ||
			action.Kind == domainrestore.Tombstone || action.Kind == domainrestore.Adopt
		isLeft := r.link.Recovery == nil || slices.ContainsFunc(r.link.Recovery.Pending, func(id []byte) bool { return bytes.Equal(id, action.FileID) })
		if isWork && isLeft {
			work = append(work, action)
		}
	}
	return work
}

func (r *recovery) execute(ctx context.Context, work []domainrestore.Action) error {
	if r.link.Recovery == nil {
		pending := make([][]byte, len(work))
		for i, action := range work {
			pending[i] = action.FileID
		}
		r.link.Recovery = &vaultlink.RecoveryProgress{Pending: pending}
		if err := r.deps.State.WriteLink(ctx, r.id, r.link); err != nil {
			return err
		}
	}
	for start := 0; start < len(work); start += batchSize {
		batch := work[start:min(start+batchSize, len(work))]
		if err := r.executeBatch(ctx, batch); err != nil {
			return err
		}
		if err := r.finish(ctx, batch); err != nil {
			return err
		}
	}
	return nil
}

func (r *recovery) finish(ctx context.Context, batch []domainrestore.Action) error {
	if err := r.deps.State.WriteRecords(ctx, r.id, r.records); err != nil {
		return err
	}
	r.link.Recovery.Pending = slices.DeleteFunc(r.link.Recovery.Pending, func(id []byte) bool {
		return slices.ContainsFunc(batch, func(action domainrestore.Action) bool { return bytes.Equal(action.FileID, id) })
	})
	return r.deps.State.WriteLink(ctx, r.id, r.link)
}

func (r *recovery) executeBatch(ctx context.Context, batch []domainrestore.Action) error {
	var files []transfer.File
	var removals []transfer.FileRecord
	for _, action := range batch {
		switch action.Kind {
		case domainrestore.Adopt:
			r.setRecord(r.getAdoptedRecord(action))
		case domainrestore.Send, domainrestore.Recreate:
			file, isThere, err := r.readFile(ctx, action)
			if err != nil {
				return err
			}
			if isThere {
				files = append(files, file)
			}
		case domainrestore.Tombstone:
			removals = append(removals, r.getRemoval(action))
		case domainrestore.Keep, domainrestore.Pull:
		}
	}
	if err := r.send(ctx, files); err != nil {
		return err
	}
	return r.remove(ctx, removals)
}

func (r *recovery) readFile(ctx context.Context, action domainrestore.Action) (transfer.File, bool, error) {
	files := r.deps.Host.GetFiles(r.id)
	stat, err := files.StatFile(ctx, action.Path)
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		return transfer.File{}, false, nil
	}
	if err != nil {
		return transfer.File{}, false, err
	}
	file := transfer.File{FileID: action.FileID, ParentRev: action.ParentRev, ParentVersion: action.ParentVersion, Path: action.Path, Content: nil, ModifiedAt: stat.ModTime.UnixNano(), Stat: nil}
	if stat.Size > note.SmallFileLimit {
		file.Stat = &stat
		return file, true, nil
	}
	content, _, err := files.Read(ctx, action.Path)
	if errors.Is(err, host.ErrNotExist) {
		return transfer.File{}, false, nil
	}
	if err != nil {
		return transfer.File{}, false, err
	}
	file.Content = content
	return file, true, nil
}

func (r *recovery) send(ctx context.Context, files []transfer.File) error {
	if len(files) == 0 {
		return nil
	}
	outcomes, err := r.deps.Commit.Execute(ctx, ports.CommitInput{VaultID: r.id, Vault: r.vault, Files: files, Shares: nil})
	if err != nil {
		return err
	}
	for _, outcome := range outcomes {
		if outcome.Status == transfer.Committed {
			r.setRecord(outcome.Record)
		}
	}
	return nil
}

func (r *recovery) remove(ctx context.Context, removals []transfer.FileRecord) error {
	if len(removals) == 0 {
		return nil
	}
	outcomes, err := r.deps.Remove.Execute(ctx, transfer.RemoveInput{Vault: r.vault, Records: removals})
	if err != nil {
		return err
	}
	for _, outcome := range outcomes {
		if outcome.Status == transfer.Committed {
			r.records = slices.DeleteFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, outcome.FileID) })
			r.link.AddRemoved(vaultlink.RemovedFile{FileID: outcome.FileID, Path: outcome.Path, HeadRev: outcome.RevID, Version: outcome.Record.Version + 1})
		}
	}
	return nil
}

func (r *recovery) setRecord(record transfer.FileRecord) {
	at := slices.IndexFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, record.FileID) })
	if at < 0 {
		r.records = append(r.records, record)
		return
	}
	r.records[at] = record
}

func (r *recovery) getKnownFiles() []domainrestore.Known {
	known := make([]domainrestore.Known, len(r.records), len(r.records)+len(r.link.Removed))
	for i, record := range r.records {
		known[i] = domainrestore.Known{FileID: record.FileID, Path: record.Path, BaseHash: record.BaseHash, HeadRev: record.HeadRev, Version: record.Version}
	}
	for _, gone := range r.link.Removed {
		_, isLocal := r.hashes[gone.Path]
		isKnown := slices.ContainsFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, gone.FileID) })
		if !isLocal && !isKnown {
			known = append(known, domainrestore.Known{FileID: gone.FileID, Path: gone.Path, BaseHash: "", HeadRev: gone.HeadRev, Version: gone.Version})
		}
	}
	return known
}

func (r *recovery) getAdoptedRecord(action domainrestore.Action) transfer.FileRecord {
	record := r.getRecord(action.FileID)
	record.FileID, record.Path, record.BaseHash, record.HeadRev, record.Version = action.FileID, action.Path, r.hashes[action.Path], action.ParentRev, action.ParentVersion
	return record
}

func (r *recovery) getRemoval(action domainrestore.Action) transfer.FileRecord {
	record := r.getRecord(action.FileID)
	record.FileID, record.Path, record.HeadRev, record.Version = action.FileID, action.Path, action.ParentRev, action.ParentVersion
	return record
}

func (r *recovery) getRecord(fileID []byte) transfer.FileRecord {
	if at := slices.IndexFunc(r.records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, fileID) }); at >= 0 {
		return r.records[at]
	}
	return transfer.NewEmptyRecord()
}
