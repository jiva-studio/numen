package syncvault

import (
	"bytes"
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
	isReset    bool
}

func (p *pass) run(ctx context.Context) error {
	if err := p.use.steps.Sweep.Sweep(ctx, p.id); err != nil {
		return err
	}
	if isPaused, err := p.use.deps.Pauses.HasPending(ctx, p.id); isPaused || err != nil {
		return err
	}
	previous, err := p.use.deps.Progress.ReadStatus(ctx, p.id)
	if err != nil {
		return err
	}
	if err := p.pull(ctx); err != nil || p.isReset {
		return err
	}
	isRetry, err := p.push(ctx)
	if err != nil {
		return p.use.reports.settle(ctx, err)
	}
	if isRetry {
		if err := p.pull(ctx); err != nil {
			return err
		}
	}
	if err := p.use.reports.report(ctx, len(p.conflicts), previous.ConflictCount, p.unverified); err != nil {
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
	p.isReset = result.IsReset
	return nil
}

func (p *pass) push(ctx context.Context) (bool, error) {
	records, err := p.use.deps.State.ReadRecords(ctx, p.id)
	if err != nil {
		return false, err
	}
	input := transfer.ScanInput{VaultID: p.id, Bases: getBases(records), Records: records}
	changes, err := p.use.steps.Scan.Execute(ctx, input)
	if err != nil {
		return false, err
	}
	scan, err := p.use.steps.Detect.Execute(ctx, input)
	if err != nil {
		return false, err
	}
	vault, err := p.openVault(ctx)
	if err != nil {
		return false, err
	}
	removals, verdict, err := p.guard(ctx, scan.Removals, len(records))
	if err != nil || verdict == transfer.Pause {
		return false, err
	}
	if verdict == transfer.Keep {
		return true, p.restore(ctx, records, removals)
	}
	isRetry := false
	for _, step := range []func(context.Context, transfer.Vault, []transfer.FileRecord) ([]transfer.FileRecord, bool, error){
		func(ctx context.Context, v transfer.Vault, r []transfer.FileRecord) ([]transfer.FileRecord, bool, error) {
			return p.sendMoves(ctx, v, r, scan.Moves)
		},
		func(ctx context.Context, v transfer.Vault, r []transfer.FileRecord) ([]transfer.FileRecord, bool, error) {
			return p.sendRemovals(ctx, v, r, removals)
		},
		func(ctx context.Context, v transfer.Vault, r []transfer.FileRecord) ([]transfer.FileRecord, bool, error) {
			return p.sendFiles(ctx, v, r, withoutMoved(changes, scan.Moves))
		},
	} {
		var retry bool
		if records, retry, err = step(ctx, vault, records); err != nil {
			return false, err
		}
		isRetry = isRetry || retry
	}
	return isRetry, p.use.deps.State.WriteRecords(ctx, p.id, records)
}

func withoutMoved(changes []transfer.Change, moves []transfer.Move) []transfer.Change {
	return slices.DeleteFunc(slices.Clone(changes), func(change transfer.Change) bool {
		return slices.ContainsFunc(moves, func(move transfer.Move) bool { return move.To == change.Path })
	})
}

func (p *pass) guard(ctx context.Context, removals []transfer.FileRecord, tracked int) ([]transfer.FileRecord, transfer.Verdict, error) {
	if len(removals) == 0 {
		return nil, transfer.Go, nil
	}
	candidates := make([]transfer.Removal, len(removals))
	for i, record := range removals {
		candidates[i] = transfer.Removal{FileID: record.FileID, Path: record.Path, BaseHash: record.BaseHash, RevID: nil}
	}
	verdict, err := p.use.steps.Guard.Execute(ctx, transfer.GuardInput{Vault: p.id, Direction: transfer.DirectionSend, Candidates: candidates, Tracked: tracked})
	return removals, verdict, err
}

func (p *pass) restore(ctx context.Context, records, removals []transfer.FileRecord) error {
	kept := slices.DeleteFunc(slices.Clone(records), func(one transfer.FileRecord) bool {
		return slices.ContainsFunc(removals, func(gone transfer.FileRecord) bool { return bytes.Equal(gone.FileID, one.FileID) })
	})
	if err := p.use.deps.State.WriteRecords(ctx, p.id, kept); err != nil {
		return err
	}
	return p.rewind(ctx)
}

func (p *pass) rewind(ctx context.Context) error {
	link, err := p.use.deps.State.ReadLink(ctx, p.id)
	if err != nil {
		return err
	}
	link.ServerSeq = 0
	return p.use.deps.State.WriteLink(ctx, p.id, link)
}

func (p *pass) sendMoves(ctx context.Context, vault transfer.Vault, records []transfer.FileRecord, moves []transfer.Move) ([]transfer.FileRecord, bool, error) {
	if len(moves) == 0 {
		return records, false, nil
	}
	outcomes, err := p.use.steps.Move.Execute(ctx, transfer.MoveInput{Vault: vault, Moves: moves})
	if err != nil {
		return records, false, err
	}
	return p.apply(records, outcomes), hasMismatch(outcomes), nil
}

func (p *pass) sendRemovals(ctx context.Context, vault transfer.Vault, records, removals []transfer.FileRecord) ([]transfer.FileRecord, bool, error) {
	if len(removals) == 0 {
		return records, false, nil
	}
	outcomes, err := p.use.steps.Remove.Execute(ctx, transfer.RemoveInput{Vault: vault, Records: removals})
	if err != nil {
		return records, false, err
	}
	for _, outcome := range outcomes {
		if outcome.Status == transfer.Committed {
			records = slices.DeleteFunc(records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, outcome.FileID) })
		}
	}
	return records, hasMismatch(outcomes), nil
}

func (p *pass) sendFiles(ctx context.Context, vault transfer.Vault, records []transfer.FileRecord, changes []transfer.Change) ([]transfer.FileRecord, bool, error) {
	files, err := p.readFiles(ctx, records, changes)
	if err != nil || len(files) == 0 {
		return records, false, err
	}
	input := transfer.CommitInput{VaultID: p.id, Vault: vault, Files: files}
	outcomes, err := p.use.steps.Commit.Execute(ctx, input)
	if err != nil {
		return records, false, err
	}
	if again := getEditsOverRemovals(files, outcomes); len(again) > 0 {
		input.Files = make([]transfer.File, len(again))
		for i, one := range again {
			input.Files[i] = one.file
		}
		second, err := p.use.steps.Commit.Execute(ctx, input)
		if err != nil {
			return records, false, err
		}
		for i, one := range again {
			outcomes[one.index] = second[i]
		}
	}
	return p.apply(records, outcomes), hasMismatch(outcomes), nil
}

// repeat is a file whose revision the server refused because the head of the file is a removal, or is gone, with the
// parent the file is committed under the second time.
type repeat struct {
	index int
	file  transfer.File
}

func getEditsOverRemovals(files []transfer.File, outcomes []transfer.Outcome) []repeat {
	var again []repeat
	for i, outcome := range outcomes {
		if outcome.Status != transfer.ParentMismatch {
			continue
		}
		file := files[i]
		switch {
		case outcome.Head == nil && len(file.ParentRev) > 0:
			file.ParentRev = nil
		case outcome.Head != nil && outcome.Head.GetRevision().GetDeleted():
			file.ParentRev = outcome.Head.GetRevId()
		default:
			continue
		}
		again = append(again, repeat{index: i, file: file})
	}
	return again
}

func hasMismatch(outcomes []transfer.Outcome) bool {
	return slices.ContainsFunc(outcomes, func(o transfer.Outcome) bool {
		return o.Status == transfer.ParentMismatch || o.Status == transfer.PathTaken
	})
}

func (p *pass) apply(records []transfer.FileRecord, outcomes []transfer.Outcome) []transfer.FileRecord {
	for _, outcome := range outcomes {
		if outcome.Status == transfer.Committed {
			records = setRecord(records, outcome.Record)
		}
	}
	return records
}

func getBases(records []transfer.FileRecord) map[string]string {
	bases := make(map[string]string, len(records))
	for _, record := range records {
		bases[record.Path] = record.BaseHash
	}
	return bases
}

func (p *pass) readFiles(ctx context.Context, records []transfer.FileRecord, changes []transfer.Change) ([]transfer.File, error) {
	var files []transfer.File
	for _, change := range changes {
		if slices.Contains(p.conflicts, change.Path) {
			continue
		}
		file := transfer.File{FileID: nil, ParentRev: nil, Path: change.Path, Content: nil, ModifiedAt: change.ModifiedAt, Stat: nil}
		isFound, err := p.readBody(ctx, &file, change.Size)
		if err != nil {
			return nil, err
		}
		if !isFound {
			continue
		}
		if at := slices.IndexFunc(records, func(one transfer.FileRecord) bool { return one.Path == change.Path }); at >= 0 {
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

func setRecord(records []transfer.FileRecord, record transfer.FileRecord) []transfer.FileRecord {
	at := slices.IndexFunc(records, func(one transfer.FileRecord) bool { return bytes.Equal(one.FileID, record.FileID) })
	if at < 0 {
		return append(records, record)
	}
	records[at] = record
	return records
}

func (p *pass) readBody(ctx context.Context, file *transfer.File, size int64) (bool, error) {
	if size > note.SmallFileLimit {
		stat, err := p.use.deps.Host.GetFiles(p.id).StatFile(ctx, file.Path)
		if err != nil {
			return false, ignoreMissing(err)
		}
		file.Stat = &stat
		return true, nil
	}
	part, err := p.use.deps.Host.GetTransfers(p.id).ReadBytes(ctx, file.Path, 0, note.SmallFileLimit)
	if err != nil {
		return false, ignoreMissing(err)
	}
	file.Content = part.Content
	return true, nil
}

func ignoreMissing(err error) error {
	if errors.Is(err, host.ErrNotExist) {
		return nil
	}
	return err
}
