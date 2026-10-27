package plugin

import (
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugindir"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// contentTypeTable gives a file the type of its name where no plugin brings one.
type contentTypeTable interface {
	GetContentType(name string) domain.ContentType
	ListExtensions(mask domain.ContentType) []string
}

// eventFilePrefix starts the event a window sends for a file it opens, followed by its extension
// or its name.
const eventFilePrefix = "onFile:"

// contentTypeCache keeps what the types of the enabled plugins come to. The text plugins are asked
// for once for each file of a vault, and listing the extensions of a mask reads the whole table.
type contentTypeCache struct {
	mu sync.Mutex
	// key names the enabled plugins and their versions the rest was computed for.
	key        string
	extensions map[domain.ContentType][]string
	opened     map[string]map[string][]string
}

// contentTypeLookup is the types the enabled plugins bring, over the table.
type contentTypeLookup struct {
	plugins []plugindir.DiscoveredPlugin
	// brought is the type each extension an enabled plugin brings gives a file, by the plugin of the
	// lowest id.
	brought map[string]domain.ContentType
	table   contentTypeTable
}

// SetContentTypeTable sets the table of the types of files. Until it is set, a file is of a type
// an enabled plugin brings, or of an unknown one.
func (h *Host) SetContentTypeTable(table contentTypeTable) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.contentTypes = table
	h.typeCache.mu.Lock()
	h.typeCache.key = ""
	h.typeCache.mu.Unlock()
}

// GetContentType is the type of a file by its name: the type an enabled plugin brings for its
// extension, and else the type the table gives it.
func (h *Host) GetContentType(name string) domain.ContentType {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getLookupLocked().getContentType(name)
}

// ListExtensions lists, lowercase, sorted and without the dot, the extensions that give a file a
// type the mask covers.
func (h *Host) ListExtensions(mask domain.ContentType) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return slices.Clone(h.listExtensionsLocked(h.getLookupLocked(), mask))
}

func (h *Host) getLookupLocked() contentTypeLookup {
	lookup := contentTypeLookup{brought: map[string]domain.ContentType{}, table: h.contentTypes}
	if !h.safeMode {
		lookup.plugins = slices.DeleteFunc(h.getPluginsInOrderLocked(), func(p plugindir.DiscoveredPlugin) bool {
			return !h.isPluginEnabledLocked(p.Manifest.ID)
		})
	}
	for _, p := range lookup.plugins {
		for _, declared := range p.Manifest.ContentTypes {
			for _, extension := range declared.ListExtensions() {
				if _, ok := lookup.brought[extension]; !ok {
					lookup.brought[extension] = declared.Type
				}
			}
		}
	}
	return lookup
}

// getKey names the enabled plugins and their versions.
func (l contentTypeLookup) getKey() string {
	var key strings.Builder
	for _, p := range l.plugins {
		key.WriteString(p.Manifest.ID + "@" + p.Manifest.Version + ",")
	}
	return key.String()
}

func (l contentTypeLookup) getContentType(name string) domain.ContentType {
	base := path.Base(name)
	if dot := strings.LastIndexByte(base, '.'); dot >= 0 {
		if found, ok := l.brought[strings.ToLower(base[dot+1:])]; ok {
			return found
		}
	}
	if l.table == nil {
		return domain.UnknownContentType
	}
	return l.table.GetContentType(name)
}

// openCacheLocked is the cache, emptied where the enabled plugins changed since it was filled. The
// caller unlocks it.
func (h *Host) openCacheLocked(lookup contentTypeLookup) *contentTypeCache {
	h.typeCache.mu.Lock()
	if key := lookup.getKey(); h.typeCache.key != key || h.typeCache.extensions == nil {
		h.typeCache.key = key
		h.typeCache.extensions = map[domain.ContentType][]string{}
		h.typeCache.opened = map[string]map[string][]string{}
	}
	return &h.typeCache
}

func (h *Host) listExtensionsLocked(lookup contentTypeLookup, mask domain.ContentType) []string {
	cache := h.openCacheLocked(lookup)
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
	for extension := range l.brought {
		candidates = append(candidates, extension)
	}
	slices.Sort(candidates)
	candidates = slices.DeleteFunc(slices.Compact(candidates), func(extension string) bool {
		return !mask.Matches(l.getContentType("file." + extension))
	})
	cache.extensions[mask] = candidates
	return candidates
}

// listOpenedExtensionsLocked lists, by reader id, the extensions of the files each reader of the
// plugin opens: those of its types where it is the default reader ChooseHandler picks among the
// readers of every enabled plugin.
func (h *Host) listOpenedExtensionsLocked(pluginID string) map[string][]string {
	lookup := h.getLookupLocked()
	cache := h.openCacheLocked(lookup)
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

// listReaderInfosLocked is each reader of the plugin with the extensions of the files it opens.
func (h *Host) listReaderInfosLocked(manifest pluginport.PluginManifest) []pluginport.PluginReaderInfo {
	if len(manifest.Readers) == 0 {
		return nil
	}
	byReader := h.listOpenedExtensionsLocked(manifest.ID)
	out := make([]pluginport.PluginReaderInfo, 0, len(manifest.Readers))
	for _, reader := range manifest.Readers {
		out = append(out, pluginport.PluginReaderInfo{ID: reader.ID, Extensions: slices.Clone(byReader[reader.ID])})
	}
	return out
}

// isOpeningFileLocked reports whether the event is a window opening a file whose type a reader or
// the text input of the plugin takes, and the plugin has a backend to start.
func (h *Host) isOpeningFileLocked(p plugindir.DiscoveredPlugin, event string) bool {
	extension, ok := strings.CutPrefix(event, eventFilePrefix)
	if !ok || extension == "" {
		return false
	}
	return p.Manifest.HasBackend() && p.Manifest.IsHandling(h.getLookupLocked().getContentType("file."+extension))
}
