package sourcebridge

import (
	"encoding/json"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
)

// DescriptionFiles keeps the description of each search source across processes.
type DescriptionFiles interface {
	ReadSourceDescription(pluginID string) ([]byte, bool)
	WriteSourceDescription(pluginID string, content []byte)
}

// ConfigRevisions names the configuration a plugin has now; equal revisions are one configuration.
type ConfigRevisions interface {
	GetPluginConfigRevision(pluginID string) string
}

// descriptionFile is the on-disk shape of the description a source gave under one revision of its
// configuration.
type descriptionFile struct {
	Revision      string  `json:"revision"`
	Name          string  `json:"name"`
	Recipe        string  `json:"recipe"`
	Width         int     `json:"width"`
	ChunkTokens   int     `json:"chunkTokens"`
	MinTextLength int     `json:"minTextLength"`
	Concurrency   int     `json:"concurrency"`
	FloorHint     float64 `json:"floorHint"`
}

// keptDescription is the description a source gave under one revision of its configuration.
type keptDescription struct {
	revision string
	read     description
}

// Descriptions keeps the last description each search source gave, by plugin id and the revision of its
// configuration, in memory and in files. A source opened under a revision kept here joins without its
// plugin being started. A description read is kept under the revision the plugin has when the read ends;
// a change of configuration during the read is read again.
type Descriptions struct {
	files     DescriptionFiles
	revisions ConfigRevisions
	mu        sync.Mutex
	byPlugin  map[string]keptDescription
}

// NewDescriptions keeps the descriptions in memory and in files, each under the revision revisions names for its
// plugin. Nil files keeps them in memory only.
func NewDescriptions(files DescriptionFiles, revisions ConfigRevisions) *Descriptions {
	return &Descriptions{files: files, revisions: revisions, byPlugin: map[string]keptDescription{}}
}

func (d *Descriptions) get(pluginID string) (description, bool) {
	if d == nil {
		return description{}, false
	}
	revision := d.revisions.GetPluginConfigRevision(pluginID)
	if revision == "" {
		return description{}, false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if kept, isKept := d.byPlugin[pluginID]; isKept && kept.revision == revision {
		return kept.read, true
	}
	kept, isRead := d.read(pluginID)
	if !isRead || kept.revision != revision {
		return description{}, false
	}
	d.byPlugin[pluginID] = kept
	return kept.read, true
}

func (d *Descriptions) put(pluginID string, read description) {
	if d == nil {
		return
	}
	revision := d.revisions.GetPluginConfigRevision(pluginID)
	if revision == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.byPlugin[pluginID] = keptDescription{revision: revision, read: read}
	if d.files == nil {
		return
	}
	content, err := json.Marshal(descriptionFile{
		Revision: revision, Name: read.source.Name, Recipe: read.source.SourceRecipe, Width: read.source.Width,
		ChunkTokens: read.source.ChunkTokens, MinTextLength: read.source.MinTextLength,
		Concurrency: read.source.Concurrency, FloorHint: read.floorHint,
	})
	if err != nil {
		return
	}
	d.files.WriteSourceDescription(pluginID, content)
}

func (d *Descriptions) read(pluginID string) (keptDescription, bool) {
	if d.files == nil {
		return keptDescription{}, false
	}
	content, isKept := d.files.ReadSourceDescription(pluginID)
	if !isKept {
		return keptDescription{}, false
	}
	var file descriptionFile
	if err := json.Unmarshal(content, &file); err != nil {
		return keptDescription{}, false
	}
	return keptDescription{revision: file.Revision, read: description{
		source: searchsource.Description{
			Name: file.Name, SourceRecipe: file.Recipe, Width: file.Width, ChunkTokens: file.ChunkTokens,
			MinTextLength: file.MinTextLength, Concurrency: file.Concurrency,
		},
		floorHint: file.FloorHint,
	}}, true
}
