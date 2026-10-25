package textversion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// errRepeatedCursor is a provider that answered a page with the cursor it was asked with.
var errRepeatedCursor = errors.New("the provider answered the same page twice")

// GetLiveText is the text of the file's live version, or textport.ErrNoText.
func (u *Versions) GetLiveText(ctx context.Context, v domain.Vault, path, fileHash string) (domain.VersionText, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	live, ok := domain.FindLiveVersion(versions, fileHash, u.isEnabled)
	if !ok {
		return domain.VersionText{}, fmt.Errorf("%w: %s has no live version", textport.ErrNoText, path)
	}
	return u.readText(ctx, v, live)
}

// GetVersionText is the text of one version, or textport.ErrNoText where the
// file has no such version or the plugin that made it is not enabled.
func (u *Versions) GetVersionText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID) (domain.VersionText, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, err
	}
	i := indexOf(versions, id)
	if i < 0 || !u.isEnabled(versions[i].PluginID) {
		return domain.VersionText{}, fmt.Errorf("%w: version %s of %s", textport.ErrNoText, id, path)
	}
	return u.readText(ctx, v, versions[i])
}

// GetTextPage is one page of a version's text, as the plugin that made it
// answers it.
func (u *Versions) GetTextPage(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (textport.TextPage, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return textport.TextPage{}, err
	}
	i := indexOf(versions, id)
	if i < 0 || !u.isEnabled(versions[i].PluginID) {
		return textport.TextPage{}, fmt.Errorf("%w: version %s of %s", textport.ErrNoText, id, path)
	}
	provider, err := u.providers.OpenProvider(ctx, versions[i].PluginID)
	if err != nil {
		return textport.TextPage{}, err
	}
	return provider.GetText(ctx, textport.TextRequest{Vault: v.ID, Path: path, Key: versions[i].Key, Cursor: cursor})
}

// readText is every page of a version's text, joined.
func (u *Versions) readText(ctx context.Context, v domain.Vault, version domain.TextVersion) (domain.VersionText, error) {
	provider, err := u.providers.OpenProvider(ctx, version.PluginID)
	if err != nil {
		return domain.VersionText{}, err
	}
	var parts []domain.TextPart
	cursor := ""
	for {
		if err := ctx.Err(); err != nil {
			return domain.VersionText{}, err
		}
		page, err := provider.GetText(ctx, textport.TextRequest{Vault: v.ID, Path: version.Path, Key: version.Key, Cursor: cursor})
		if err != nil {
			return domain.VersionText{}, fmt.Errorf("read text %s of %s: %w", version.GetID(), version.Path, err)
		}
		parts = append(parts, page.Parts...)
		if page.NextCursor == "" {
			return domain.NewVersionText(version, parts), nil
		}
		if page.NextCursor == cursor {
			return domain.VersionText{}, fmt.Errorf("read text %s of %s: %w", version.GetID(), version.Path, errRepeatedCursor)
		}
		cursor = page.NextCursor
	}
}

// Open is the text of one vault, as the index, the search and the highlighting read it.
func (u *Versions) Open(v domain.Vault) (textport.VaultTexts, error) {
	return vaultTexts{versions: u, vault: v}, nil
}

type vaultTexts struct {
	versions *Versions
	vault    domain.Vault
}

func (t vaultTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	listing, err := t.versions.ListVersions(ctx, t.vault, path, fileHash)
	if err != nil {
		return "", err
	}
	if listing.Live == "" {
		return "", fmt.Errorf("%w: %s has no live version", textport.ErrNoText, path)
	}
	return listing.Live, nil
}

func (t vaultTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	return t.versions.GetLiveText(ctx, t.vault, path, fileHash)
}

func (t vaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	return t.versions.GetVersionText(ctx, t.vault, path, id)
}

func (t vaultTexts) RequestText(ctx context.Context, path, fileHash string) error {
	return t.versions.RequestText(ctx, t.vault, path, fileHash)
}
