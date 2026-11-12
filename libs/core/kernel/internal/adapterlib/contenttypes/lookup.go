package contenttypes

import (
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Table gives a file the type of its name.
type Table interface {
	GetContentType(name string) domain.ContentType
	ListExtensions(mask domain.ContentType) []string
}

// cache stores the resolved types of the enabled plugins. The text plugins are queried once for each
// file of a vault, and listing the extensions of a mask reads the whole table.
type cache struct {
	mu         sync.Mutex
	key        string
	extensions map[domain.ContentType][]string
	opened     map[string]map[string][]string
}

// Lookup gives a file the type of its name, as the table of types says, and keeps what it worked out
// until the set of enabled plugins changes.
type Lookup struct {
	mu    sync.RWMutex
	table Table
	cache cache
}

// SetTable sets the table of the types of files. Until it is set, a file is of an unknown type.
func (l *Lookup) SetTable(table Table) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.table = table
	l.cache.mu.Lock()
	l.cache.key = ""
	l.cache.mu.Unlock()
}

func (l *Lookup) getTable() Table {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.table
}

// GetContentType is the type of a file by its name, as the table gives it.
func (l *Lookup) GetContentType(name string) domain.ContentType {
	table := l.getTable()
	if table == nil {
		return domain.UnknownContentType
	}
	return table.GetContentType(name)
}

func getKey(plugins []manifest.Discovered) string {
	var key strings.Builder
	for _, p := range plugins {
		key.WriteString(p.Manifest.ID + "@" + p.Manifest.Version + ",")
	}
	return key.String()
}

func (l *Lookup) openCache(plugins []manifest.Discovered) *cache {
	l.cache.mu.Lock()
	if key := getKey(plugins); l.cache.key != key || l.cache.extensions == nil {
		l.cache.key = key
		l.cache.extensions = map[domain.ContentType][]string{}
		l.cache.opened = map[string]map[string][]string{}
	}
	return &l.cache
}

// ListExtensions lists, lowercase, sorted and without the dot, the extensions that give a file a type
// the mask covers, over the enabled plugins.
func (l *Lookup) ListExtensions(plugins []manifest.Discovered, mask domain.ContentType) []string {
	cache := l.openCache(plugins)
	defer cache.mu.Unlock()
	return slices.Clone(l.listCachedTypes(cache, mask))
}

func (l *Lookup) listCachedTypes(cache *cache, mask domain.ContentType) []string {
	if found, ok := cache.extensions[mask]; ok {
		return found
	}
	var candidates []string
	if table := l.getTable(); table != nil {
		candidates = table.ListExtensions(mask)
	}
	slices.Sort(candidates)
	candidates = slices.DeleteFunc(slices.Compact(candidates), func(extension string) bool {
		return !mask.Matches(l.GetContentType("file." + extension))
	})
	cache.extensions[mask] = candidates
	return candidates
}

func (l *Lookup) listOpenedTypes(plugins []manifest.Discovered, pluginID string) map[string][]string {
	cache := l.openCache(plugins)
	defer cache.mu.Unlock()
	if found, ok := cache.opened[pluginID]; ok {
		return found
	}
	var handlers []domain.ContentHandler
	for _, p := range plugins {
		handlers = append(handlers, p.Manifest.ListContentHandlers()...)
	}
	var extensions []string
	for _, one := range handlers {
		if one.PluginID == pluginID {
			extensions = append(extensions, l.listCachedTypes(cache, one.Type)...)
		}
	}
	slices.Sort(extensions)
	out := map[string][]string{}
	for _, extension := range slices.Compact(extensions) {
		chosen, ok := domain.ChooseHandler(l.GetContentType("file."+extension), handlers)
		if ok && chosen.PluginID == pluginID {
			out[chosen.ID] = append(out[chosen.ID], extension)
		}
	}
	cache.opened[pluginID] = out
	return out
}

// ListReaderInfos lists the readers a plugin declares with the extensions each opens, over the
// enabled plugins.
func (l *Lookup) ListReaderInfos(plugins []manifest.Discovered, declared manifest.Manifest) []plugindomain.ReaderInfo {
	if len(declared.Contributes.Readers) == 0 {
		return nil
	}
	byReader := l.listOpenedTypes(plugins, declared.ID)
	out := make([]plugindomain.ReaderInfo, 0, len(declared.Contributes.Readers))
	for _, reader := range declared.Contributes.Readers {
		out = append(out, plugindomain.ReaderInfo{ID: reader.ID, Extensions: slices.Clone(byReader[reader.ID])})
	}
	return out
}
