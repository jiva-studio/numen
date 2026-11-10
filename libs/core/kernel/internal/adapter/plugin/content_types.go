package plugin

import (
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// contentTypeTable gives a file the type of its name.
type contentTypeTable interface {
	GetContentType(name string) domain.ContentType
	ListExtensions(mask domain.ContentType) []string
}

// contentTypeCache stores the resolved types of the enabled plugins. The text
// plugins are queried once for each file of a vault, and listing the extensions
// of a mask reads the whole table.
type contentTypeCache struct {
	mu         sync.Mutex
	key        string
	extensions map[domain.ContentType][]string
	opened     map[string]map[string][]string
}

// contentTypeLookup is the enabled plugins over the table.
type contentTypeLookup struct {
	plugins []manifest.Discovered
	table   contentTypeTable
}

// ContentTypes gives a file the type of its name, as the table of types says, over the enabled
// plugins, and keeps what it worked out until the set of enabled plugins changes.
type ContentTypes struct {
	t     *table
	mu    sync.RWMutex
	table contentTypeTable
	cache contentTypeCache
}

func newContentTypes(t *table) *ContentTypes {
	return &ContentTypes{t: t}
}

// SetContentTypeTable sets the table of the types of files. Until it is set, a file is of an
// unknown type.
func (c *ContentTypes) SetContentTypeTable(table contentTypeTable) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.table = table
	c.cache.mu.Lock()
	c.cache.key = ""
	c.cache.mu.Unlock()
}

// GetContentType is the type of a file by its name, as the table gives it.
func (c *ContentTypes) GetContentType(name string) domain.ContentType {
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	return c.getLookupLocked().getContentType(name)
}

// ListExtensions lists, lowercase, sorted and without the dot, the extensions that give a file a
// type the mask covers.
func (c *ContentTypes) ListExtensions(mask domain.ContentType) []string {
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	return slices.Clone(c.listExtensionsLocked(c.getLookupLocked(), mask))
}

func (c *ContentTypes) getLookupLocked() contentTypeLookup {
	c.mu.RLock()
	lookup := contentTypeLookup{table: c.table}
	c.mu.RUnlock()
	if !c.t.safeMode {
		lookup.plugins = slices.DeleteFunc(c.t.getPluginsInOrderLocked(), func(p manifest.Discovered) bool {
			return !c.t.isPluginEnabledLocked(p.Manifest.ID)
		})
	}
	return lookup
}

func (l contentTypeLookup) getKey() string {
	var key strings.Builder
	for _, p := range l.plugins {
		key.WriteString(p.Manifest.ID + "@" + p.Manifest.Version + ",")
	}
	return key.String()
}

func (l contentTypeLookup) getContentType(name string) domain.ContentType {
	if l.table == nil {
		return domain.UnknownContentType
	}
	return l.table.GetContentType(name)
}

func (c *ContentTypes) openCacheLocked(lookup contentTypeLookup) *contentTypeCache {
	c.cache.mu.Lock()
	if key := lookup.getKey(); c.cache.key != key || c.cache.extensions == nil {
		c.cache.key = key
		c.cache.extensions = map[domain.ContentType][]string{}
		c.cache.opened = map[string]map[string][]string{}
	}
	return &c.cache
}

func (c *ContentTypes) listExtensionsLocked(lookup contentTypeLookup, mask domain.ContentType) []string {
	cache := c.openCacheLocked(lookup)
	defer cache.mu.Unlock()
	return lookup.listExtensionsCached(cache, mask)
}

func (l contentTypeLookup) listExtensionsCached(cache *contentTypeCache, mask domain.ContentType) []string {
	if found, ok := cache.extensions[mask]; ok {
		return found
	}
	var candidates []string
	if l.table != nil {
		candidates = l.table.ListExtensions(mask)
	}
	slices.Sort(candidates)
	candidates = slices.DeleteFunc(slices.Compact(candidates), func(extension string) bool {
		return !mask.Matches(l.getContentType("file." + extension))
	})
	cache.extensions[mask] = candidates
	return candidates
}

func (c *ContentTypes) listOpenedExtensionsLocked(pluginID string) map[string][]string {
	lookup := c.getLookupLocked()
	cache := c.openCacheLocked(lookup)
	defer cache.mu.Unlock()
	if found, ok := cache.opened[pluginID]; ok {
		return found
	}
	var handlers []domain.ContentHandler
	for _, p := range lookup.plugins {
		handlers = append(handlers, p.Manifest.ListContentHandlers()...)
	}
	var extensions []string
	for _, one := range handlers {
		if one.PluginID == pluginID {
			extensions = append(extensions, lookup.listExtensionsCached(cache, one.Type)...)
		}
	}
	slices.Sort(extensions)
	out := map[string][]string{}
	for _, extension := range slices.Compact(extensions) {
		chosen, ok := domain.ChooseHandler(lookup.getContentType("file."+extension), handlers)
		if ok && chosen.PluginID == pluginID {
			out[chosen.ID] = append(out[chosen.ID], extension)
		}
	}
	cache.opened[pluginID] = out
	return out
}

func (c *ContentTypes) listReaderInfosLocked(manifest manifest.Manifest) []plugindomain.ReaderInfo {
	if len(manifest.Contributes.Readers) == 0 {
		return nil
	}
	byReader := c.listOpenedExtensionsLocked(manifest.ID)
	out := make([]plugindomain.ReaderInfo, 0, len(manifest.Contributes.Readers))
	for _, reader := range manifest.Contributes.Readers {
		out = append(out, plugindomain.ReaderInfo{ID: reader.ID, Extensions: slices.Clone(byReader[reader.ID])})
	}
	return out
}
