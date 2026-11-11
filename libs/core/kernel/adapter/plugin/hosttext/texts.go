package hosttext

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// Texts is the text versions of the vaults, for the plugins that call the host. Each call works
// the vault it is given. Which version is live is decided for the file as it is now.
type Texts struct {
	versions *textversion.Versions
}

// NewTexts returns the text versions of the vaults.
func NewTexts(versions *textversion.Versions) Texts {
	return Texts{versions: versions}
}

// PutVersion records a version a plugin made of a file of the vault.
func (p Texts) PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error) {
	return p.versions.PutVersion(ctx, v, pluginID, params)
}

// RemoveVersion removes a version a plugin made of a file of the vault.
func (p Texts) RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error {
	return p.versions.RemoveVersion(ctx, v, pluginID, path, key)
}

// ListVersions lists the versions of a file of the vault and names the live one for the file's
// current content.
func (p Texts) ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error) {
	return p.versions.Texts.ListFileVersions(ctx, v, path)
}

// ReadText reads one page of the segments of a version of a file of the vault.
func (p Texts) ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error) {
	return p.versions.Texts.GetTextPage(ctx, v, path, id, cursor)
}

// ReadTextRange reads the segments of a version of a file of the vault that cover a range of its
// whole text.
func (p Texts) ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error) {
	return p.versions.Texts.GetTextRange(ctx, v, path, id, start, length)
}
