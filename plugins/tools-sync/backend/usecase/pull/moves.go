package pull

import (
	"context"
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

// applyMove moves the local file to the new path of a metadata-only revision. It reports whether the revision is
// dealt with; a file that is not there to move is left to the download.
func (r *run) applyMove(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, record transfer.FileRecord, localAtNew string) (bool, error) {
	if localAtNew == meta.Hash {
		return true, r.adopt(ctx, stored, meta, record, true)
	}
	oldHash, oldStat, err := r.readLocal(ctx, record.Path)
	if err != nil || oldHash == "" {
		return false, err
	}
	if oldHash != record.BaseHash || localAtNew != "" {
		r.addConflict(meta.Path, stored.GetSeq())
		return true, nil
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

type folderMove struct{ from, to string }

// applyFolderMoves moves a folder whole when the page moves every file under it to one other folder, so the folder
// never shows half of itself.
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

// getFolderMove strips the longest common tail of segments from both paths.
func getFolderMove(from, to string) folderMove {
	a, b := strings.Split(from, "/"), strings.Split(to, "/")
	for len(a) > 1 && len(b) > 1 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	return folderMove{from: path.Join(a...), to: path.Join(b...)}
}

func (r *run) moveFolder(ctx context.Context, group folderMove, moved []transfer.FileRecord) error {
	under := 0
	for _, record := range r.records {
		if strings.HasPrefix(record.Path, group.from+"/") {
			under++
		}
	}
	if len(moved) < 2 || len(moved) != under {
		return nil
	}
	for _, record := range moved {
		hash, _, err := r.readLocal(ctx, record.Path)
		if err != nil || hash == "" {
			return err
		}
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
		if strings.HasPrefix(record.Path, group.from+"/") {
			r.records[i].Path = path.Join(group.to, strings.TrimPrefix(record.Path, group.from+"/"))
		}
	}
	return nil
}
