package version

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// IndexVersions is the text versions the index database records, one vault at a time.
type IndexVersions struct {
	repo *Repository
}

// NewIndexVersions wraps a Repository as a textport.VersionStores.
func NewIndexVersions(r *Repository) IndexVersions {
	return IndexVersions{repo: r}
}

func (r IndexVersions) OpenVersionReader(v domain.Vault) (textport.VersionReader, error) {
	return vaultVersions{repo: r.repo, vault: v.ID}, nil
}

func (r IndexVersions) OpenVersionWriter(v domain.Vault) (textport.VersionWriter, error) {
	return vaultVersions{repo: r.repo, vault: v.ID}, nil
}

func (r IndexVersions) OpenContentHashes(v domain.Vault) (textport.ContentHashes, error) {
	return vaultVersions{repo: r.repo, vault: v.ID}, nil
}

type vaultVersions struct {
	repo  *Repository
	vault domain.VaultID
}

func (r vaultVersions) GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error) {
	return r.repo.GetVersions(ctx, r.vault, path)
}

func (r vaultVersions) GetVersionsUnder(ctx context.Context, path string) ([]domain.TextVersion, error) {
	return r.repo.GetVersionsUnder(ctx, r.vault, path)
}

func (r vaultVersions) SaveVersion(ctx context.Context, v domain.TextVersion) error {
	return r.repo.SaveVersion(ctx, r.vault, v)
}

func (r vaultVersions) RemoveVersion(ctx context.Context, path string, id domain.TextVersionID) error {
	pluginID, key, _ := strings.Cut(string(id), "/")
	return r.repo.RemoveVersion(ctx, r.vault, path, pluginID, key)
}

func (r vaultVersions) RemovePluginVersions(ctx context.Context, pluginID string) ([]domain.TextVersion, error) {
	return r.repo.RemovePluginVersions(ctx, r.vault, pluginID)
}

func (r vaultVersions) FindContentHash(ctx context.Context, path string) (text.ContentHash, bool, error) {
	return r.repo.FindContentHash(ctx, r.vault, path)
}

func (r vaultVersions) SaveContentHash(ctx context.Context, record text.ContentHash) error {
	return r.repo.SaveContentHash(ctx, r.vault, record)
}
