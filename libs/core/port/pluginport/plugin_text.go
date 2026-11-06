package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// PluginTextPut is what a plugin reports of one of its text versions. File is
// the name of the version's file in the plugin's folder.
type PluginTextPut struct {
	Path      string
	Key       string
	Source    domain.TextVersionID
	Kind      domain.TextKind
	InputHash string
	File      string
}

// PluginTextListing is the versions of a file and the live one, which is empty
// where none is live.
type PluginTextListing struct {
	Versions []domain.TextVersion
	Live     domain.TextVersionID
}

// PluginTextPage is one page of a version's segments. An empty NextCursor is the last page.
type PluginTextPage struct {
	Segments   []domain.TextSegment
	NextCursor string
}

// PluginTextRange is a ranged read of a version: the segments, where the first begins in the whole
// text, and the bytes of the whole text.
type PluginTextRange struct {
	Segments []domain.TextSegment
	Start    int
	Size     int
}

// PluginTexts is the text versions of the vaults, as the plugins that call the
// host work them. Each call works the vault it is given.
type PluginTexts interface {
	// PutVersion records a version of the plugin, or replaces the one it has under the key.
	PutVersion(ctx context.Context, v domain.Vault, pluginID string, params PluginTextPut) (domain.TextVersion, error)
	// RemoveVersion removes the record of a version of the plugin.
	RemoveVersion(ctx context.Context, v domain.Vault, pluginID, path, key string) error
	// ListVersions is the versions of a file and which of them is live.
	ListVersions(ctx context.Context, v domain.Vault, path string) (PluginTextListing, error)
	// ReadText is one page of any version's segments, read from its file.
	ReadText(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, cursor string) (PluginTextPage, error)
	// ReadTextRange is the segments of a version that cover length bytes of its whole text from
	// start, each cut to the range at whole characters. A length of zero reads to the end.
	ReadTextRange(ctx context.Context, v domain.Vault, path string, id domain.TextVersionID, start, length int) (PluginTextRange, error)
}

// PluginTextHost receives the text versions that plugins work on.
type PluginTextHost interface {
	// SetPluginTexts is the text versions the plugins work. Nil is none.
	SetPluginTexts(texts PluginTexts)
}
