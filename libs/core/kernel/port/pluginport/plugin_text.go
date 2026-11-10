package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginTexts is the text versions of the vaults, as the plugins that call the
// host work them. Each call works the vault it is given.
type PluginTexts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params plugin.TextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, v domain.Vault, path string) (plugin.TextListing, error)
	// ReadText is one page of any version's segments, read from its file.
	ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (plugin.TextPage, error)
	// ReadTextRange returns the segments of a version that cover length bytes of its whole text from
	// start, each truncated to the range at whole characters. A length of zero reads to the end.
	ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (plugin.TextRange, error)
}

// PluginTextHost receives the text versions that plugins work on.
type PluginTextHost interface {
	// SetPluginTexts is the text versions the plugins work. Nil is none.
	SetPluginTexts(texts PluginTexts)
}
