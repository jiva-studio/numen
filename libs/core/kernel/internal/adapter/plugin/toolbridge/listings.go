package toolbridge

import (
	"encoding/json"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
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

// listing is the answer one version of a plugin gave to ListTools.
type listing struct {
	version string
	tools   *v1.ListToolsResponse
}

// listings keeps the ListTools answer of every plugin by plugin id and version, in memory and in files.
type listings struct {
	files    ListingFiles
	mu       sync.Mutex
	byPlugin map[string]listing
}

func newListings(files ListingFiles) *listings {
	return &listings{files: files, byPlugin: map[string]listing{}}
}

// get returns the tools the version of the plugin listed, and false when it has not been listed.
func (l *listings) get(pluginID, version string) (*v1.ListToolsResponse, bool) {
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

// put keeps the tools the version of the plugin listed.
func (l *listings) put(pluginID, version string, tools *v1.ListToolsResponse) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byPlugin[pluginID] = listing{version: version, tools: tools}
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

// read returns the listing kept in the file of the plugin; a file that cannot be decoded counts as none.
func (l *listings) read(pluginID string) (listing, bool) {
	content, isKept := l.files.ReadToolListing(pluginID)
	if !isKept {
		return listing{}, false
	}
	var file listingFile
	if err := json.Unmarshal(content, &file); err != nil {
		return listing{}, false
	}
	tools := &v1.ListToolsResponse{}
	if err := protojson.Unmarshal(file.Listing, tools); err != nil {
		return listing{}, false
	}
	return listing{version: file.Version, tools: tools}, true
}
