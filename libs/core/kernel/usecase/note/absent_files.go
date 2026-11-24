package note

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// WithAbsent returns the removal that also deletes the entries of absent files at the path and under it.
func (u Remove) WithAbsent(absent vaultport.AbsentFiles) Remove {
	u.absent = absent
	return u
}

// WithAbsent returns the move that also moves the entries of absent files at the path and under it.
func (u MoveFile) WithAbsent(absent vaultport.AbsentFiles) MoveFile {
	u.absent = absent
	return u
}

func listAbsentUnder(ctx context.Context, absent vaultport.AbsentFiles, v domain.Vault, path string) ([]vault.AbsentFile, error) {
	if absent == nil {
		return nil, nil
	}
	all, err := absent.ListAbsentFiles(ctx, v)
	if err != nil {
		return nil, err
	}
	var under []vault.AbsentFile
	for _, entry := range all {
		if entry.Path == path || strings.HasPrefix(entry.Path, path+"/") {
			under = append(under, entry)
		}
	}
	return under, nil
}

func (u Remove) removeAbsent(ctx context.Context, v domain.Vault, path string) (int, error) {
	under, err := listAbsentUnder(ctx, u.absent, v, path)
	if err != nil || len(under) == 0 {
		return 0, err
	}
	paths := make([]string, len(under))
	for i, entry := range under {
		paths[i] = entry.Path
	}
	return len(paths), u.absent.RemoveAbsentFiles(ctx, v, paths)
}

func (u MoveFile) moveAbsent(ctx context.Context, v domain.Vault, from, to string) (int, error) {
	under, err := listAbsentUnder(ctx, u.absent, v, from)
	if err != nil || len(under) == 0 {
		return 0, err
	}
	moved := make([]vault.AbsentFile, len(under))
	paths := make([]string, len(under))
	for i, entry := range under {
		paths[i] = entry.Path
		entry.Path = getMovedPath(from, to, entry.Path)
		moved[i] = entry
	}
	if err := u.absent.RemoveAbsentFiles(ctx, v, paths); err != nil {
		return 0, err
	}
	return len(moved), u.absent.PutAbsentFiles(ctx, v, moved)
}
