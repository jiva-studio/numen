package hosttest

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	_ host.FileMover       = (*Vault)(nil)
	_ host.FileRangeReader = (*Vault)(nil)
	_ host.FilePlacer      = (*Vault)(nil)
)

type fileMove struct{ from, target string }

// MoveFile files a file, or a folder with everything under it, somewhere else. It repairs no link.
func (v *Vault) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (host.MoveFileResult, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := errors.Join(v.check(ctx, from), v.check(ctx, to)); err != nil {
		return host.MoveFileResult{}, err
	}
	sources, isFolder, err := v.listEntry(from)
	if err != nil {
		return host.MoveFileResult{}, err
	}
	moves := make([]fileMove, 0, len(sources))
	for _, source := range sources {
		target := to
		if isFolder {
			target = path.Join(to, strings.TrimPrefix(source, from+"/"))
		}
		moves = append(moves, fileMove{from: source, target: target})
	}
	if err := v.checkMoves(moves, baseline, isFolder); err != nil {
		return host.MoveFileResult{}, err
	}
	for _, move := range moves {
		if err := v.moveFile(move.from, move.target); err != nil {
			return host.MoveFileResult{}, err
		}
	}
	return host.MoveFileResult{Count: len(moves)}, nil
}

func (v *Vault) checkMoves(moves []fileMove, baseline *domain.FileStat, isFolder bool) error {
	if !isFolder {
		if err := v.checkBaseline(moves[0].from, baseline); err != nil {
			return err
		}
	}
	for _, move := range moves {
		if v.isPresent(move.target) {
			return fmt.Errorf("%s: %w", move.target, host.ErrAlreadyExists)
		}
	}
	return nil
}

func (v *Vault) checkBaseline(filePath string, baseline *domain.FileStat) error {
	if baseline == nil {
		return nil
	}
	now, err := v.files.stat(filePath)
	if err != nil {
		return err
	}
	isStale, err := v.isStale(now, *baseline)
	if err != nil {
		return err
	}
	if isStale {
		return host.ErrStale
	}
	return nil
}

// listEntry returns the file at the path, or the files under the folder at it.
func (v *Vault) listEntry(entry string) (files []string, isFolder bool, err error) {
	if v.isPresent(entry) {
		return []string{entry}, false, nil
	}
	paths, err := v.files.paths()
	if err != nil {
		return nil, false, err
	}
	for _, filePath := range paths {
		if strings.HasPrefix(filePath, entry+"/") {
			files = append(files, filePath)
		}
	}
	if len(files) == 0 {
		return nil, false, host.ErrNotExist
	}
	return files, true, nil
}

// RemoveFile moves a file, or a folder with everything under it, to the trash. The trash is the folder
// ".trash" of the vault, and Trashed lists the paths of the trash.
func (v *Vault) RemoveFile(ctx context.Context, filePath string, baseline *domain.FileStat) (host.RemoveFileResult, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return host.RemoveFileResult{}, err
	}
	files, isFolder, err := v.listEntry(filePath)
	if err != nil {
		return host.RemoveFileResult{}, err
	}
	if !isFolder {
		if err := v.checkBaseline(filePath, baseline); err != nil {
			return host.RemoveFileResult{}, err
		}
	}
	for _, file := range files {
		if err := v.moveFile(file, v.getTrashPath(file)); err != nil {
			return host.RemoveFileResult{}, err
		}
		v.trashed = append(v.trashed, file)
	}
	return host.RemoveFileResult{Count: len(files)}, nil
}

// Trashed lists the paths the vault had before RemoveFile moved them to the trash, in the order of the removals.
func (v *Vault) Trashed() []string {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	return slices.Clone(v.trashed)
}

func (v *Vault) getTrashPath(filePath string) string {
	target := path.Join(trashDir, filePath)
	for attempt := 2; v.isPresent(target); attempt++ {
		extension := path.Ext(filePath)
		target = path.Join(trashDir, strings.TrimSuffix(filePath, extension)+"-"+strconv.Itoa(attempt)+extension)
	}
	return target
}

// ReadBytes reads at most length bytes of any file from start, as they are.
func (v *Vault) ReadBytes(ctx context.Context, filePath string, start, length int64) (host.ByteRange, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return host.ByteRange{}, err
	}
	if start < 0 || length < 1 || length > host.MaxReadBytes {
		return host.ByteRange{}, host.ErrInvalidRange
	}
	at, err := v.files.stat(filePath)
	if err != nil {
		return host.ByteRange{}, err
	}
	content, err := v.files.read(filePath)
	if err != nil {
		return host.ByteRange{}, err
	}
	stat, err := v.hashed(at)
	if err != nil {
		return host.ByteRange{}, err
	}
	if start >= int64(len(content)) {
		return host.ByteRange{Content: []byte{}, Stat: stat}, nil
	}
	end := min(start+length, int64(len(content)))
	return host.ByteRange{Content: content[start:end], Stat: stat}, nil
}

// PlaceFile moves a file of the store folder to a path of the vault.
func (v *Vault) PlaceFile(ctx context.Context, name, filePath, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, filePath); err != nil {
		return domain.FileStat{}, err
	}
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return domain.FileStat{}, host.ErrOutside
	}
	staged := v.getFolder()
	content, isStaged := staged.files[name]
	switch {
	case !isStaged:
		return domain.FileStat{}, host.ErrNotExist
	case domain.HashContent(content) != expectedHash:
		return domain.FileStat{}, host.ErrHashMismatch
	case baseline == nil && v.isPresent(filePath):
		return domain.FileStat{}, host.ErrAlreadyExists
	}
	if err := v.checkBaseline(filePath, baseline); err != nil {
		return domain.FileStat{}, err
	}
	stat, err := v.put(filePath, content)
	if err != nil {
		return domain.FileStat{}, err
	}
	delete(staged.files, name)
	v.rescan(filePath)
	return stat, nil
}
