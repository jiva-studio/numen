package pull

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/layer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

func (r *run) isLazy(meta note.Meta, isListed bool) bool {
	if isListed {
		return true
	}
	if !r.deps.LazyHeavy {
		return false
	}
	return layer.ClassifyLayer(meta.Path, meta.Size) == layer.OnTap
}

func (r *run) getListedHash(ctx context.Context, filePath string) (string, error) {
	if r.listed == nil {
		files, err := r.deps.Host.GetAbsent(r.in.VaultID).ListAbsentFiles(ctx)
		if err != nil {
			return "", err
		}
		r.listed = make(map[string]string, len(files))
		for _, file := range files {
			r.listed[file.Path] = file.Hash
		}
	}
	hash, isListed := r.listed[filePath]
	if !isListed {
		return "", fmt.Errorf("read %s: %w", filePath, host.ErrFileAbsent)
	}
	return hash, nil
}

func (r *run) list(ctx context.Context, stored *syncv1.StoredRevision, meta note.Meta, previous transfer.FileRecord, local string, isListed, isKnown bool) error {
	absent := r.deps.Host.GetAbsent(r.in.VaultID)
	if local != "" && !isListed {
		if err := absent.EvictFile(ctx, meta.Path, local); err != nil {
			return r.holdOnRefusal(err, host.ErrChanged, meta.Path, stored.GetSeq())
		}
	}
	entry := host.AbsentFile{Path: meta.Path, Size: meta.Size, ModTime: time.Unix(0, meta.MTime), Hash: meta.Hash}
	if err := absent.PutAbsentFiles(ctx, []host.AbsentFile{entry}); err != nil {
		return r.holdOnRefusal(err, host.ErrFilePresent, meta.Path, stored.GetSeq())
	}
	if r.listed != nil {
		r.listed[meta.Path] = meta.Hash
	}
	if isKnown && previous.Path != meta.Path {
		if err := r.dropListed(ctx, previous.Path); err != nil {
			return err
		}
	}
	record, err := r.makeRecord(ctx, stored, meta, previous, isKnown)
	if err != nil {
		return err
	}
	r.setRecord(record)
	r.result.Applied = append(r.result.Applied, meta.Path)
	return nil
}

func (r *run) holdOnRefusal(err, refusal error, filePath string, seq uint64) error {
	if errors.Is(err, refusal) {
		r.addConflict(filePath, seq)
		return nil
	}
	return err
}

func (r *run) dropListed(ctx context.Context, filePath string) error {
	if _, err := r.getListedHash(ctx, filePath); errors.Is(err, host.ErrFileAbsent) {
		return nil
	} else if err != nil {
		return err
	}
	if _, err := r.moves.RemoveFile(ctx, filePath, nil); err != nil && !errors.Is(err, host.ErrNotExist) {
		return err
	}
	delete(r.listed, filePath)
	return nil
}
