package plugin

import (
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/contenttypes"
)

// enabledPlugins lists the plugins that are enabled.
type enabledPlugins interface {
	listEnabled() []manifest.Discovered
}

// ContentTypes gives a file the type of its name, as the table of types says, over the enabled
// plugins.
type ContentTypes struct {
	set    enabledPlugins
	lookup *contenttypes.Lookup
}

func newContentTypes(set enabledPlugins, lookup *contenttypes.Lookup) *ContentTypes {
	return &ContentTypes{set: set, lookup: lookup}
}

// SetContentTypeTable sets the table of the types of files. Until it is set, a file is of an
// unknown type.
func (c *ContentTypes) SetContentTypeTable(table contenttypes.Table) {
	c.lookup.SetTable(table)
}

// GetContentType is the type of a file by its name, as the table gives it.
func (c *ContentTypes) GetContentType(name string) domain.ContentType {
	return c.lookup.GetContentType(name)
}

// ListExtensions lists, lowercase, sorted and without the dot, the extensions that give a file a
// type the mask covers.
func (c *ContentTypes) ListExtensions(mask domain.ContentType) []string {
	return c.lookup.ListExtensions(c.set.listEnabled(), mask)
}
