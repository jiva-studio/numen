package plugin

import (
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/contenttypes"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ContentTypes gives a file the type of its name, as the table of types says, over the enabled
// plugins.
type ContentTypes struct {
	t      *table
	lookup contenttypes.Lookup
}

func newContentTypes(t *table) *ContentTypes {
	return &ContentTypes{t: t}
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
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	return c.lookup.ListExtensions(c.listEnabledLocked(), mask)
}

func (c *ContentTypes) listEnabledLocked() []manifest.Discovered {
	if c.t.safeMode {
		return nil
	}
	return slices.DeleteFunc(c.t.getPluginsInOrderLocked(), func(p manifest.Discovered) bool {
		return !c.t.isPluginEnabledLocked(p.Manifest.ID)
	})
}

func (c *ContentTypes) listReaderInfosLocked(declared manifest.Manifest) []plugindomain.ReaderInfo {
	return c.lookup.ListReaderInfos(c.listEnabledLocked(), declared)
}
