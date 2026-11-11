package textversion_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// folderStores is the plugin folders of memoryFiles, each a plugin store.
type folderStores struct{ files *memoryFiles }

func (s folderStores) Open(_ domain.Vault, pluginID string) (pluginport.PluginStoreReader, error) {
	return folderStore{files: s.files, pluginID: pluginID}, nil
}

func (s folderStores) DeleteFolder(_ context.Context, _ domain.Vault, pluginID string) error {
	s.files.mu.Lock()
	defer s.files.mu.Unlock()
	for key := range s.files.files {
		if strings.HasPrefix(key, pluginID+"/") {
			delete(s.files.files, key)
		}
	}
	return nil
}

type folderStore struct {
	pluginport.PluginStoreReader
	files    *memoryFiles
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

func TestDeletePluginDataRemovesTheFolderAndTheRecords(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.runEntry(t, ocr)
	s.listener.take()
	data, err := textversion.NewPluginData(s.versions.Versions, folderStores{files: s.folders})
	if err != nil {
		t.Fatal(err)
	}
	size, err := data.GetPluginData(t.Context(), s.vault, ocr)
	if err != nil || size.FileCount != 1 || size.ByteCount == 0 {
		t.Fatalf("the recognizer's folder holds %+v (%v), want one file", size, err)
	}

	if err := data.DeletePluginData(t.Context(), s.vault, ocr); err != nil {
		t.Fatal(err)
	}

	if size, err := data.GetPluginData(t.Context(), s.vault, ocr); err != nil || size.FileCount != 0 {
		t.Fatalf("after the delete the recognizer's folder holds %+v (%v)", size, err)
	}
	listing, err := s.versions.Texts.ListVersions(t.Context(), s.vault, scanPath, s.getContentHash())
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Versions[0].PluginID != pdfReader || listing.Live != s.getReaderVersion() {
		t.Fatalf("after the delete the scan has %+v, want the reader's version alone and live", listing)
	}
	if size, err := data.GetPluginData(t.Context(), s.vault, pdfReader); err != nil || size.FileCount != 1 {
		t.Fatalf("the reader's folder holds %+v (%v), want its file kept", size, err)
	}
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Fatalf("deleting the data reported %v, want one change of %s", got, scanPath)
	}
}

func TestNewPluginDataRequiresVersionsAndStores(t *testing.T) {
	if _, err := textversion.NewPluginData(nil, folderStores{}); err == nil {
		t.Fatal("plugin data was made without text versions")
	}
}
