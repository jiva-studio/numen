package hosttest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var _ host.AbsentFiles = (*Vault)(nil)

// PutAbsentFiles lists the files, replacing the entry of each path. A path that holds a file of another
// hash returns host.ErrFilePresent and none is listed. A path that holds the same content is not listed.
// The feed reports nothing for a listed file.
func (v *Vault) PutAbsentFiles(ctx context.Context, files []host.AbsentFile) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return err
	}
	var listed []host.AbsentFile
	for _, file := range files {
		content, err := v.files.read(file.Path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			listed = append(listed, file)
		case err != nil:
			return err
		case !strings.EqualFold(domain.HashContent(content), file.Hash):
			return fmt.Errorf("list %s as absent: %w", file.Path, host.ErrFilePresent)
		}
	}
	if v.absent == nil {
		v.absent = map[string]host.AbsentFile{}
	}
	for _, file := range listed {
		v.absent[file.Path] = file
	}
	return nil
}

// ListAbsentFiles returns the files listed, by path.
func (v *Vault) ListAbsentFiles(ctx context.Context) ([]host.AbsentFile, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.gate(ctx); err != nil {
		return nil, err
	}
	listed := make([]host.AbsentFile, 0, len(v.absent))
	for _, file := range v.absent {
		listed = append(listed, file)
	}
	slices.SortFunc(listed, func(a, b host.AbsentFile) int { return strings.Compare(a.Path, b.Path) })
	return listed, nil
}

// FillFile places the store file at the path of a listed file once its SHA-256 is the hash of the entry.
// The feed reports nothing for the fill.
func (v *Vault) FillFile(ctx context.Context, path, storeName string) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, path); err != nil {
		return err
	}
	entry, isListed := v.absent[path]
	if !isListed {
		return fmt.Errorf("fill %s: not listed as absent: %w", path, host.ErrNotExist)
	}
	content, isStored := v.getFolder().files[storeName]
	if !isStored {
		return fmt.Errorf("fill %s: the store has no %s: %w", path, storeName, host.ErrNotExist)
	}
	if !strings.EqualFold(domain.HashContent(content), entry.Hash) {
		return fmt.Errorf("fill %s: %w", path, host.ErrHashMismatch)
	}
	if _, err := v.put(path, content); err != nil {
		return err
	}
	delete(v.absent, path)
	return nil
}

// EvictFile deletes the bytes of the file and lists it, when its SHA-256 is expectedHash. The feed reports
// no removal.
func (v *Vault) EvictFile(ctx context.Context, path, expectedHash string) error {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.check(ctx, path); err != nil {
		return err
	}
	content, err := v.files.read(path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(domain.HashContent(content), expectedHash) {
		return fmt.Errorf("drop %s: %w", path, host.ErrChanged)
	}
	stat, err := v.files.stat(path)
	if err != nil {
		return err
	}
	if v.absent == nil {
		v.absent = map[string]host.AbsentFile{}
	}
	v.absent[path] = host.AbsentFile{Path: path, Size: stat.Size, ModTime: stat.ModTime, Hash: domain.HashContent(content)}
	return v.files.remove(path)
}

func (v *Vault) checkAbsent(path string, readErr error) error {
	if _, isListed := v.absent[path]; isListed && errors.Is(readErr, fs.ErrNotExist) {
		return host.ErrFileAbsent
	}
	return readErr
}

func (v *Vault) isListed(filePath string) bool {
	_, isListed := v.absent[filePath]
	return isListed
}

func (v *Vault) listEntryPaths() ([]string, error) {
	paths, err := v.files.listPaths()
	if err != nil {
		return nil, err
	}
	for listed := range v.absent {
		paths = append(paths, listed)
	}
	return paths, nil
}

func (v *Vault) moveEntry(from, target string) error {
	entry, isListed := v.absent[from]
	if !isListed {
		return v.moveFile(from, target)
	}
	if v.isPresent(target) || v.isListed(target) {
		return fmt.Errorf("%s: %w", target, host.ErrAlreadyExists)
	}
	delete(v.absent, from)
	entry.Path = target
	v.absent[target] = entry
	return nil
}
