package toolbridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/jiva-studio/numen/libs/atomicfile"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

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

// listings keeps the ListTools answer of every plugin by plugin id and version, in memory and, while dir
// names a folder, in one file per plugin there.
type listings struct {
	dir      func() string
	mu       sync.Mutex
	byPlugin map[string]listing
}

func newListings(dir func() string) *listings {
	return &listings{dir: dir, byPlugin: map[string]listing{}}
}

// get returns the tools the version of the plugin listed, and false when it has not been listed.
func (l *listings) get(pluginID, version string) (*v1.ListToolsResponse, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if kept, isKept := l.byPlugin[pluginID]; isKept && kept.version == version {
		return kept.tools, true
	}
	read, err := l.read(pluginID)
	if err != nil || read.version != version {
		return nil, false
	}
	l.byPlugin[pluginID] = read
	return read.tools, true
}

// put keeps the tools the version of the plugin listed. A file that cannot be written leaves the
// listing in memory only, and the plugin is asked again in the next process.
func (l *listings) put(pluginID, version string, tools *v1.ListToolsResponse) {
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := listing{version: version, tools: tools}
	l.byPlugin[pluginID] = kept
	_ = l.write(pluginID, kept)
}

func (l *listings) getPath(pluginID string) (string, error) {
	dir := l.dir()
	if dir == "" {
		return "", errors.New("no folder keeps the tool listings")
	}
	if pluginID == "" || strings.HasPrefix(pluginID, ".") || strings.ContainsAny(pluginID, `/\`) {
		return "", fmt.Errorf("plugin id %q does not name a file", pluginID)
	}
	return filepath.Join(dir, pluginID+".json"), nil
}

func (l *listings) read(pluginID string) (listing, error) {
	path, err := l.getPath(pluginID)
	if err != nil {
		return listing{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return listing{}, err
	}
	var file listingFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return listing{}, err
	}
	tools := &v1.ListToolsResponse{}
	if err := protojson.Unmarshal(file.Listing, tools); err != nil {
		return listing{}, err
	}
	return listing{version: file.Version, tools: tools}, nil
}

func (l *listings) write(pluginID string, kept listing) error {
	path, err := l.getPath(pluginID)
	if err != nil {
		return err
	}
	encoded, err := protojson.Marshal(kept.tools)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(listingFile{Version: kept.version, Listing: encoded})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(path, bytes.NewReader(raw), 0o600)
}
