package pull

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

func (r *run) applyMove(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, record transfer.FileRecord, localAtNew string, newStat domain.FileStat) (bool, error) {
	if localAtNew == meta.Hash {
		return true, r.adopt(ctx, stored, meta, record, true)
	}
	oldHash, oldStat, err := r.readLocal(ctx, record.Path)
	if err != nil || oldHash == "" {
		return false, err
	}
	if localAtNew != "" {
		return true, r.resolveTaken(ctx, stored, meta, localAtNew, newStat, moveSource{record: record, stat: oldStat})
	}
	if _, err := r.moves.MoveFile(ctx, record.Path, meta.Path, &oldStat); err != nil {
		if errors.Is(err, host.ErrStale) || errors.Is(err, host.ErrAlreadyExists) {
			r.addConflict(meta.Path, stored.GetSeq())
			return true, nil
		}
		return true, err
	}
	r.result.Applied = append(r.result.Applied, meta.Path)
	return true, r.adopt(ctx, stored, meta, record, true)
}

func (r *run) resolveTaken(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, local string, stat domain.FileStat, from moveSource) error {
	if err := r.resolve(ctx, stored, meta, local, stat); err != nil {
		return err
	}
	if moved, isKnown := r.getRecord(from.record.FileID); !isKnown || moved.Path != meta.Path || from.stat.Hash != from.record.BaseHash {
		return nil
	}
	_, err := r.moves.RemoveFile(ctx, from.record.Path, &from.stat)
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrStale) {
		return nil
	}
	return err
}

type moveSource struct {
	record transfer.FileRecord
	stat   domain.FileStat
}

type folderMove struct{ from, to string }

func (r *run) applyFolderMoves(ctx context.Context, page []*syncv1.StoredRevision) error {
	groups := map[folderMove][]transfer.FileRecord{}
	for _, stored := range page {
		meta, checkErr := r.check(stored)
		record, isKnown := r.getRecord(stored.GetRevision().GetFileId())
		if checkErr != nil || !isKnown || stored.GetRevision().GetDeleted() || record.Path == meta.Path || !isMetadataOnly(stored.GetRevision(), record, meta) {
			continue
		}
		group := getFolderMove(record.Path, meta.Path)
		groups[group] = append(groups[group], record)
	}
	for group, moved := range groups {
		if err := r.moveFolder(ctx, group, moved); err != nil {
			return err
		}
	}
	return nil
}

func getFolderMove(from, to string) folderMove {
	a, b := strings.Split(from, "/"), strings.Split(to, "/")
	for len(a) > 1 && len(b) > 1 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	return folderMove{from: path.Join(a...), to: path.Join(b...)}
}

func (r *run) moveFolder(ctx context.Context, group folderMove, moved []transfer.FileRecord) error {
	if len(moved) < 2 || len(moved) != r.countInFolder(group.from) {
		return nil
	}
	if isPresent, err := r.hasAllLocal(ctx, moved); !isPresent || err != nil {
		return err
	}
	if _, err := r.moves.MoveFile(ctx, group.from, group.to, nil); err != nil {
		if errors.Is(err, host.ErrAlreadyExists) || errors.Is(err, host.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, record := range moved {
		r.result.Applied = append(r.result.Applied, path.Join(group.to, strings.TrimPrefix(record.Path, group.from+"/")))
	}
	r.records = slices.Clone(r.records)
	for i, record := range r.records {
		if rest, isUnder := strings.CutPrefix(record.Path, group.from+"/"); isUnder {
			r.records[i].Path = path.Join(group.to, rest)
		}
	}
	return nil
}

func (r *run) countInFolder(folder string) int {
	count := 0
	for _, record := range r.records {
		if strings.HasPrefix(record.Path, folder+"/") {
			count++
		}
	}
	return count
}

func (r *run) hasAllLocal(ctx context.Context, moved []transfer.FileRecord) (bool, error) {
	for _, record := range moved {
		hash, _, err := r.readLocal(ctx, record.Path)
		if err != nil || hash == "" {
			return false, err
		}
	}
	return true, nil
}
