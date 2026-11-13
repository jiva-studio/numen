package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/atomicfile"
)

// toolsFolder is the folder inside the cache root where the host keeps the tools each plugin listed.
const toolsFolder = "tools"

// ReadToolListing returns the tool listing kept for the plugin, and false where none is kept or the host
// has no cache root.
func (s *Storage) ReadToolListing(pluginID string) ([]byte, bool) {
	path := s.getToolListingPath(pluginID)
	if path == "" {
		return nil, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.report(fmt.Errorf("read the tools of plugin %s: %w", pluginID, err))
		}
		return nil, false
	}
	return content, true
}

// WriteToolListing keeps the tool listing of the plugin, whole. Where it cannot, the failure is reported
// and the plugin is asked for its tools again in the next process.
func (s *Storage) WriteToolListing(pluginID string, content []byte) {
	path := s.getToolListingPath(pluginID)
	if path == "" {
		return
	}
	if err := createPrivateDir(filepath.Dir(path)); err != nil {
		s.report(fmt.Errorf("prepare the tools directory: %w", err))
		return
	}
	if err := atomicfile.WriteFile(path, bytes.NewReader(content), 0o600); err != nil {
		s.report(fmt.Errorf("keep the tools of plugin %s: %w", pluginID, err))
	}
}

func (s *Storage) getToolListingPath(pluginID string) string {
	s.mu.RLock()
	root := s.roots.cache
	s.mu.RUnlock()
	if root == "" || checkStorageID(pluginID) != nil {
		return ""
	}
	return filepath.Join(root, toolsFolder, pluginID+".json")
}
