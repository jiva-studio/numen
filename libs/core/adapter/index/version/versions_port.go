package version

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// IndexVersions is the text versions the index database records, one vault at a time.
type IndexVersions struct {
	repo *Repository
}

// NewIndexVersions wraps a Repository as a textport.VersionRepositories.
func NewIndexVersions(r *Repository) IndexVersions {
	return IndexVersions{repo: r}
}

func (r IndexVersions) Open(v domain.Vault) (textport.VersionRepository, error) {
	return vaultVersions{repo: r.repo, vault: v.ID}, nil
}

type vaultVersions struct {
	repo  *Repository
	vault domain.VaultID
}

func (r vaultVersions) GetVersions(ctx context.Context, path string) ([]domain.TextVersion, error) {
	return r.repo.GetVersions(ctx, r.vault, path)
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
