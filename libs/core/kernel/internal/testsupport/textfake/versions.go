package textfake

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/textport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// TextVersions are the use cases over the text versions of vault files, built the way the container builds them:
// the three work the same stores.
type TextVersions struct {
	*textversion.Versions
	Texts   *textversion.Texts
	Changes *textversion.Changes
}

// NewTextVersions builds the use cases over the stores, the file stores, the enabled plugins, the vault
// readers and the clock.
func NewTextVersions(
	stores textport.VersionStores, files textport.TextFileStores, plugins textport.TextPlugins,
	readers vaultport.VaultReaders, clock machineport.Clock,
) *TextVersions {
	texts := textversion.NewTexts(stores, files, readers, note.NewReadFrontmatter(readers), clock)
	changes := textversion.NewChanges(texts)
	versions := textversion.NewVersions(stores, files, plugins, clock, changes)
	return &TextVersions{Versions: versions, Texts: texts, Changes: changes}
}
