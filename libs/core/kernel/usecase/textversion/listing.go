package textversion

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
)

// live finds the live version of a file among its versions.
type live struct {
	files textport.TextFileStores
	notes noteport.NoteReader
}

func (u live) find(
	ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion, contentHash string,
) (domain.TextVersion, bool, error) {
	if len(versions) == 0 {
		return domain.TextVersion{}, false, nil
	}
	properties, err := u.readSourceProperties(ctx, v, path, versions)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	files, err := u.files.Open(v)
	if err != nil {
		return domain.TextVersion{}, false, err
	}
	found, ok := domain.FindLiveVersion(versions, contentHash, properties, func(version domain.TextVersion) bool {
		file, err := files.ReadTextFile(ctx, version.PluginID, version.File)
		return err == nil && file.TextHash == version.TextHash
	})
	return found, ok, nil
}

func (u live) readSourceProperties(
	ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion,
) (map[string]any, error) {
	if !slices.ContainsFunc(versions, domain.TextVersion.IsFromProperty) {
		return nil, nil
	}
	_, properties, err := u.notes.Execute(ctx, v, path)
	if err != nil {
		return nil, fmt.Errorf("read the properties of %s: %w", path, err)
	}
	return properties, nil
}

func getVersions(ctx context.Context, stores textport.VersionStores, v domain.Vault, path string) ([]domain.TextVersion, error) {
	reader, err := stores.OpenVersionReader(v)
	if err != nil {
		return nil, err
	}
	return reader.GetVersions(ctx, path)
}

// ListVersions returns the versions of a file and the live one, evaluated for
// the file content identified by contentHash.
func (u *Texts) ListVersions(ctx context.Context, v domain.Vault, path, contentHash string) (plugin.TextListing, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil {
		return plugin.TextListing{}, err
	}
	listing := plugin.TextListing{Versions: versions}
	live, ok, err := u.live.find(ctx, v, path, versions, contentHash)
	if err != nil {
		return plugin.TextListing{}, err
	}
	if ok {
		listing.Live = live.GetID()
	}
	return listing, nil
}

// ListFileVersions lists the versions of a file of the vault and names the live one for the file's
// current content.
func (u *Texts) ListFileVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error) {
	contentHash, err := u.GetContentHash(ctx, v, path)
	if errors.Is(err, text.ErrFileUnreadable) {
		return plugin.TextListing{}, fmt.Errorf("%w: %w", text.ErrNoText, err)
	}
	if err != nil {
		return plugin.TextListing{}, err
	}
	return u.ListVersions(ctx, v, path, contentHash)
}

// ListTargetVersions lists the versions of the file at the path as the when variable target.versions
// holds them, each marked valid where its source chain still matches the file, and the live one. The
// file is read for its content hash only where it has versions.
func (u *Texts) ListTargetVersions(ctx context.Context, v domain.Vault, path string) ([]plugin.TargetVersion, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	if err != nil || len(versions) == 0 {
		return []plugin.TargetVersion{}, err
	}
	contentHash, err := u.GetContentHash(ctx, v, path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	properties, err := u.live.readSourceProperties(ctx, v, path, versions)
	if err != nil {
		return nil, err
	}
	live, hasLive, err := u.live.find(ctx, v, path, versions, contentHash)
	if err != nil {
		return nil, err
	}
	out := make([]plugin.TargetVersion, 0, len(versions))
	for _, one := range versions {
		out = append(out, plugin.TargetVersion{
			Kind: string(one.Kind), PluginID: one.PluginID, IsLive: hasLive && one.GetID() == live.GetID(),
			IsValid: domain.IsChainValid(versions, one.GetID(), contentHash, properties),
		})
	}
	return out, nil
}

// IsTextSource reports whether the index keeps the text of the file: it has text versions.
func (u *Texts) IsTextSource(ctx context.Context, v domain.Vault, path string) (bool, error) {
	versions, err := getVersions(ctx, u.stores, v, path)
	return len(versions) > 0, err
}
