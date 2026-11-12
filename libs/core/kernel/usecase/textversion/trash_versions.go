package textversion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// TrashVersions deletes the text versions of files that are gone for good.
type TrashVersions struct {
	stores textport.VersionStores
	files  textport.TextFileStores
}

func NewTrashVersions(stores textport.VersionStores, files textport.TextFileStores) TrashVersions {
	return TrashVersions{stores: stores, files: files}
}

// DeleteUnder deletes the records and the files of the versions of everything
// under a folder, except those of the paths in kept and of what lies under them.
// A version that cannot be deleted does not stop the others; the failures are joined.
func (u TrashVersions) DeleteUnder(ctx context.Context, v domain.Vault, folder string, kept []string) error {
	reader, err := u.stores.OpenVersionReader(v)
	if err != nil {
		return err
	}
	versions, err := reader.GetVersionsUnder(ctx, folder)
	if err != nil {
		return err
	}
	writer, err := u.stores.OpenVersionWriter(v)
	if err != nil {
		return err
	}
	files, err := u.files.Open(v)
	if err != nil {
		return err
	}
	var failed []error
	for _, one := range versions {
		if isKept(one.Path, kept) {
			continue
		}
		if err := writer.RemoveVersion(ctx, one.Path, one.GetID()); err != nil {
			failed = append(failed, fmt.Errorf("remove version %s of %s: %w", one.GetID(), one.Path, err))
			continue
		}
		if err := files.DeleteTextFile(ctx, one.PluginID, one.File); err != nil {
			failed = append(failed, fmt.Errorf("delete the file %s of %s: %w", one.File, one.PluginID, err))
		}
	}
	return errors.Join(failed...)
}

func isKept(path string, kept []string) bool {
	for _, k := range kept {
		if path == k || strings.HasPrefix(path, k+"/") {
			return true
		}
	}
	return false
}
