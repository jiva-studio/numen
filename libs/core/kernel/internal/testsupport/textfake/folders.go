package textfake

import (
	"context"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// PluginFolders is the folders of Folders, each opened as the store of a plugin.
type PluginFolders struct{ Files *Folders }

func (s PluginFolders) Open(_ domain.Vault, pluginID string) (pluginport.PluginStoreReader, error) {
	return folderStore{files: s.Files, pluginID: pluginID}, nil
}

func (s PluginFolders) DeleteFolder(_ context.Context, _ domain.Vault, pluginID string) error {
	s.Files.mu.Lock()
	defer s.Files.mu.Unlock()
	for key := range s.Files.files {
		if strings.HasPrefix(key, pluginID+"/") {
			delete(s.Files.files, key)
		}
	}
	return nil
}

type folderStore struct {
	pluginport.PluginStoreReader
	files    *Folders
	pluginID string
}

func (s folderStore) List(context.Context) ([]plugin.StoreFile, error) {
	s.files.mu.Lock()
	defer s.files.mu.Unlock()
	var out []plugin.StoreFile
	for key, raw := range s.files.files {
		if name, ok := strings.CutPrefix(key, s.pluginID+"/"); ok {
			out = append(out, plugin.StoreFile{Name: name, Size: int64(len(raw))})
		}
	}
	return out, nil
}
