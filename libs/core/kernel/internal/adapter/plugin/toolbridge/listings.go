package toolbridge

import (
	"encoding/json"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

// ListingFiles keeps the tool listing of each plugin across processes.
type ListingFiles interface {
	ReadToolListing(pluginID string) ([]byte, bool)
	WriteToolListing(pluginID string, content []byte)
}

// listingFile is the on-disk shape of the tools one version of a plugin listed.
type listingFile struct {
	Version string          `json:"version"`
	Listing json.RawMessage `json:"listing"`
}

// toolList is the answer one version of a plugin gave to ListTools.
type toolList struct {
	version string
	tools   *v1.ListToolsResponse
}

// listings keeps the ListTools answer of every plugin by plugin id, version and revision of the catalog of
// search functions the tool descriptions name, in memory and in files.
type listings struct {
	files       ListingFiles
	getRevision func() string
	mu          sync.Mutex
	byPlugin    map[string]toolList
}

func newListings(files ListingFiles, getRevision func() string) *listings {
	return &listings{files: files, getRevision: getRevision, byPlugin: map[string]toolList{}}
}

func (l *listings) getKey(version string) string {
	return version + "+" + l.getRevision()
}

func (l *listings) get(pluginID, version string) (*v1.ListToolsResponse, bool) {
	version = l.getKey(version)
	l.mu.Lock()
	defer l.mu.Unlock()
	if kept, isKept := l.byPlugin[pluginID]; isKept && kept.version == version {
		return kept.tools, true
	}
	read, isRead := l.read(pluginID)
	if !isRead || read.version != version {
		return nil, false
	}
	l.byPlugin[pluginID] = read
	return read.tools, true
}

func (l *listings) put(pluginID, version string, tools *v1.ListToolsResponse) {
	version = l.getKey(version)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byPlugin[pluginID] = toolList{version: version, tools: tools}
	encoded, err := protojson.Marshal(tools)
	if err != nil {
		return
	}
	content, err := json.Marshal(listingFile{Version: version, Listing: encoded})
	if err != nil {
		return
	}
	l.files.WriteToolListing(pluginID, content)
}

func (l *listings) read(pluginID string) (toolList, bool) {
	content, isKept := l.files.ReadToolListing(pluginID)
	if !isKept {
		return toolList{}, false
	}
	var file listingFile
	if err := json.Unmarshal(content, &file); err != nil {
		return toolList{}, false
	}
	tools := &v1.ListToolsResponse{}
	if err := protojson.Unmarshal(file.Listing, tools); err != nil {
		return toolList{}, false
	}
	return toolList{version: file.Version, tools: tools}, true
}

// ListKeptTools returns the tools the plugin last listed, read from the files and not from the plugin,
// and nil where none are kept.
func ListKeptTools(files ListingFiles, pluginID string) []*v1.PluginTool {
	read, isRead := newListings(files, func() string { return "" }).read(pluginID)
	if !isRead {
		return nil
	}
	return read.tools.GetTools()
}

// ListKeptItems returns the lister of the kept tools of a plugin as the items its contribution shows: each
// tool by its title, or by its name where it has no title, with its name as the detail.
func ListKeptItems(files ListingFiles) func(pluginID string) []plugindomain.ContributionItem {
	return func(pluginID string) []plugindomain.ContributionItem {
		kept := ListKeptTools(files, pluginID)
		items := make([]plugindomain.ContributionItem, 0, len(kept))
		for _, one := range kept {
			text := one.GetTitle()
			if text == "" {
				text = one.GetName()
			}
			items = append(items, plugindomain.ContributionItem{Text: text, Detail: one.GetName()})
		}
		return items
	}
}
