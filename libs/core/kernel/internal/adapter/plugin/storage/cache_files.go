package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
)

// The folders inside the cache root where the host keeps what plugins answered, one file per plugin.
const (
	// toolsFolder holds the tools each plugin listed.
	toolsFolder = "tools"
	// sourcesFolder holds the description each search source gave.
	sourcesFolder = "sources"
	// connectionsFolder holds the connections each plugin detected.
	connectionsFolder = "connections"
	// optionListsFolder holds the options each plugin listed for the row fields of its connections.
	optionListsFolder = "models"
)

func listCacheFolders() []string {
	return []string{toolsFolder, sourcesFolder, connectionsFolder, optionListsFolder}
}

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

// ReadDetectedConnections returns the detected connections kept for the plugin, and false where none are
// kept or the host has no cache root.
func (s *Storage) ReadDetectedConnections(pluginID string) ([]byte, bool) {
	return s.readCacheFile(connectionsFolder, pluginID, "the detected connections")
}

// WriteDetectedConnections keeps the detected connections of the plugin, whole. Where it cannot, the
// failure is reported and the plugin is asked for its connections again in the next process.
func (s *Storage) WriteDetectedConnections(pluginID string, content []byte) {
	s.writeCacheFile(connectionsFolder, pluginID, "the detected connections", content)
}

// optionListFile is the on-disk shape of one option list.
type optionListFile struct {
	ConnectionID   string            `json:"connectionId"`
	Field          string            `json:"field"`
	RowOptions     map[string]string `json:"rowOptions"`
	Operations     []string          `json:"operations"`
	Revision       int               `json:"revision"`
	PluginVersion  string            `json:"pluginVersion"`
	ConfigRevision string            `json:"configRevision"`
	GrantRevision  string            `json:"grantRevision"`
	Fields         map[string]string `json:"fields"`
	Options        []optionFile      `json:"options"`
}

type optionFile struct {
	Value string `json:"value"`
	Title string `json:"title,omitempty"`
	Group string `json:"group,omitempty"`
}

// ReadOptionLists returns the option lists kept for the connections of the plugin, and false where none
// are kept, the file cannot be read or the host has no cache root.
func (s *Storage) ReadOptionLists(pluginID string) ([]connection.OptionList, bool) {
	content, isKept := s.readCacheFile(optionListsFolder, pluginID, "the option lists")
	if !isKept {
		return nil, false
	}
	var files []optionListFile
	if err := json.Unmarshal(content, &files); err != nil {
		s.report(fmt.Errorf("read the option lists of plugin %s: %w", pluginID, err))
		return nil, false
	}
	lists := make([]connection.OptionList, len(files))
	for i, one := range files {
		options := make([]modelcall.Option, len(one.Options))
		for j, option := range one.Options {
			options[j] = modelcall.Option{Value: option.Value, Title: option.Title, Group: option.Group}
		}
		lists[i] = connection.OptionList{
			ConnectionID: one.ConnectionID, Field: one.Field, RowOptions: one.RowOptions, Operations: one.Operations,
			Revision: one.Revision, PluginVersion: one.PluginVersion, ConfigRevision: one.ConfigRevision,
			GrantRevision: one.GrantRevision, Fields: one.Fields, Options: options,
		}
	}
	return lists, true
}

// WriteOptionLists keeps the option lists of the connections of the plugin, whole, in place of those kept
// before. Where it cannot, the failure is reported.
func (s *Storage) WriteOptionLists(pluginID string, lists []connection.OptionList) {
	files := make([]optionListFile, len(lists))
	for i, one := range lists {
		options := make([]optionFile, len(one.Options))
		for j, option := range one.Options {
			options[j] = optionFile{Value: option.Value, Title: option.Title, Group: option.Group}
		}
		files[i] = optionListFile{
			ConnectionID: one.ConnectionID, Field: one.Field, RowOptions: one.RowOptions, Operations: one.Operations,
			Revision: one.Revision, PluginVersion: one.PluginVersion, ConfigRevision: one.ConfigRevision,
			GrantRevision: one.GrantRevision, Fields: one.Fields, Options: options,
		}
	}
	content, err := json.Marshal(files)
	if err != nil {
		s.report(fmt.Errorf("keep the option lists of plugin %s: %w", pluginID, err))
		return
	}
	s.writeCacheFile(optionListsFolder, pluginID, "the option lists", content)
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
