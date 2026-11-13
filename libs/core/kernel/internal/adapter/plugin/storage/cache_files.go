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

// The folders inside the cache root where the host keeps what plugins answered, one file per plugin.
const (
	// toolsFolder holds the tools each plugin listed.
	toolsFolder = "tools"
	// sourcesFolder holds the description each search source gave.
	sourcesFolder = "sources"
)

// ReadToolListing returns the tool listing kept for the plugin, and false where none is kept or the host
// has no cache root.
func (s *Storage) ReadToolListing(pluginID string) ([]byte, bool) {
	return s.readCacheFile(toolsFolder, pluginID, "the tools")
}

// WriteToolListing keeps the tool listing of the plugin, whole. Where it cannot, the failure is reported
// and the plugin is asked for its tools again in the next process.
func (s *Storage) WriteToolListing(pluginID string, content []byte) {
	s.writeCacheFile(toolsFolder, pluginID, "the tools", content)
}

// ReadSourceDescription returns the search source description kept for the plugin, and false where none is
// kept or the host has no cache root.
func (s *Storage) ReadSourceDescription(pluginID string) ([]byte, bool) {
	return s.readCacheFile(sourcesFolder, pluginID, "the search source description")
}

// WriteSourceDescription keeps the search source description of the plugin, whole. Where it cannot, the
// failure is reported and the plugin is asked for its description again in the next process.
func (s *Storage) WriteSourceDescription(pluginID string, content []byte) {
	s.writeCacheFile(sourcesFolder, pluginID, "the search source description", content)
}

func (s *Storage) readCacheFile(folder, pluginID, what string) ([]byte, bool) {
	path := s.getCacheFilePath(folder, pluginID)
	if path == "" {
		return nil, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			s.report(fmt.Errorf("read %s of plugin %s: %w", what, pluginID, err))
		}
		return nil, false
	}
	return content, true
}

func (s *Storage) writeCacheFile(folder, pluginID, what string, content []byte) {
	path := s.getCacheFilePath(folder, pluginID)
	if path == "" {
		return
	}
	if err := createPrivateDir(filepath.Dir(path)); err != nil {
		s.report(fmt.Errorf("prepare the %s directory: %w", folder, err))
		return
	}
	if err := atomicfile.WriteFile(path, bytes.NewReader(content), 0o600); err != nil {
		s.report(fmt.Errorf("keep %s of plugin %s: %w", what, pluginID, err))
	}
}

func (s *Storage) getCacheFilePath(folder, pluginID string) string {
	s.mu.RLock()
	root := s.roots.cache
	s.mu.RUnlock()
	if root == "" || checkStorageID(pluginID) != nil {
		return ""
	}
	return filepath.Join(root, folder, pluginID+".json")
}
