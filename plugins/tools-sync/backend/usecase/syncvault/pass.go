package syncvault

import (
	"context"
	"encoding/hex"
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/trust"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type pass struct {
	use        *SyncVault
	id         domain.VaultID
	authors    trust.Authors
	conflicts  []string
	unverified int
}

func (p *pass) run(ctx context.Context) error {
	if err := p.pull(ctx); err != nil {
		return err
	}
	isRetry, err := p.push(ctx)
	if err != nil {
		return p.use.reports.settle(ctx, p.id, err)
	}
	if isRetry {
		if err := p.pull(ctx); err != nil {
			return err
		}
	}
	if err := p.use.reports.report(ctx, p.id, len(p.conflicts), p.unverified); err != nil {
		return err
	}
	return p.use.deps.Progress.WriteStatus(ctx, p.id, ports.Status{LastSyncedAt: p.use.deps.Clock.Now(), ConflictCount: len(p.conflicts)})
}

func (p *pass) pull(ctx context.Context) error {
	result, err := p.use.steps.Pull.Execute(ctx, transfer.PullInput{VaultID: p.id, IsAuthor: p.authors.IsTrusted})
	if err != nil {
		return err
	}
	p.conflicts = result.Conflicts
	p.unverified = result.Unverified
	return nil
}

func (p *pass) push(ctx context.Context) (bool, error) {
	records, err := p.use.deps.State.ReadRecords(ctx, p.id)
	if err != nil {
		return false, err
	}
	changes, err := p.use.steps.Scan.Execute(ctx, transfer.ScanInput{VaultID: p.id, Bases: getBases(records)})
	if err != nil {
		return false, err
	}
	files, err := p.readFiles(ctx, records, changes)
	if err != nil || len(files) == 0 {
		return false, err
	}
	vault, err := p.openVault(ctx)
	if err != nil {
		return false, err
	}
	outcomes, err := p.use.steps.Commit.Execute(ctx, transfer.CommitInput{Vault: vault, Files: files})
	if err != nil {
		return false, err
	}
	return p.record(ctx, records, files, outcomes)
}

func getBases(records []note.FileRecord) map[string]string {
	bases := make(map[string]string, len(records))
	for _, record := range records {
		bases[record.Path] = record.BaseHash
	}
	return bases
}

func (p *pass) readFiles(ctx context.Context, records []note.FileRecord, changes []transfer.Change) ([]transfer.File, error) {
	var files []transfer.File
	reader := p.use.deps.Host.GetFiles(p.id)
	for _, change := range changes {
		if slices.Contains(p.conflicts, change.Path) {
			continue
		}
		content, _, err := reader.Read(ctx, change.Path)
		if errors.Is(err, host.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		file := transfer.File{FileID: nil, ParentRev: nil, Path: change.Path, Content: content, ModifiedAt: change.ModifiedAt}
		if at := slices.IndexFunc(records, func(one note.FileRecord) bool { return one.Path == change.Path }); at >= 0 {
			file.FileID, file.ParentRev = records[at].FileID, records[at].HeadRev
		}
		files = append(files, file)
	}
	return files, nil
}

func (p *pass) openVault(ctx context.Context) (transfer.Vault, error) {
	link, err := p.use.deps.State.ReadLink(ctx, p.id)
	if err != nil {
		return transfer.Vault{}, err
	}
	remote, err := hex.DecodeString(link.RemoteVaultID)
	if err != nil {
		return transfer.Vault{}, err
	}
	key, err := p.use.deps.Crypto.OpenData(ctx, vaultlink.GetStateKeyContext(remote), link.SealedKey)
	if err != nil {
		return transfer.Vault{}, err
	}
	return transfer.Vault{ID: remote, Key: [32]byte(key)}, nil
}

func (p *pass) record(ctx context.Context, records []note.FileRecord, files []transfer.File, outcomes []transfer.Outcome) (bool, error) {
	isRetry := false
	for i, outcome := range outcomes {
		switch outcome.Status {
		case transfer.Committed:
			records = setRecord(records, note.FileRecord{
				FileID: outcome.FileID, Path: outcome.Path, BaseHash: note.Hash(files[i].Content), HeadRev: outcome.RevID,
			})
		case transfer.ParentMismatch, transfer.PathTaken:
			isRetry = true
		case transfer.TooLarge:
		}
	}
	return isRetry, p.use.deps.State.WriteRecords(ctx, p.id, records)
}

func setRecord(records []note.FileRecord, record note.FileRecord) []note.FileRecord {
	at := slices.IndexFunc(records, func(one note.FileRecord) bool { return one.Path == record.Path })
	if at < 0 {
		return append(records, record)
	}
	records[at] = record
	return records
}
