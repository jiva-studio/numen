package storage_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
)

const finchPlugin = "org.example.finch"

func newStorage(t *testing.T, cacheRoot string, reported *[]error) *storage.Storage {
	t.Helper()
	files := storage.New(func(err error) { *reported = append(*reported, err) }, func(string) string { return "1.0.0" })
	files.SetStorageRoots(cacheRoot, "")
	return files
}

func TestTheDetectedConnectionsOfAPluginAreStoredInTheConnectionsFolder(t *testing.T) {
	root := t.TempDir()
	var reported []error
	files := newStorage(t, root, &reported)

	files.WriteDetectedConnections(finchPlugin, []byte(`{"answers":{}}`))

	content, isKept := newStorage(t, root, &reported).ReadDetectedConnections(finchPlugin)
	if !isKept || string(content) != `{"answers":{}}` || len(reported) != 0 {
		t.Errorf("content = %q, kept = %v, reported = %v", content, isKept, reported)
	}
	if _, err := os.Stat(filepath.Join(root, "connections", finchPlugin+".json")); err != nil {
		t.Errorf("the file of the plugin: %v", err)
	}
}

func TestTheOptionListsOfAPluginAreStoredInTheModelsFolder(t *testing.T) {
	root := t.TempDir()
	var reported []error
	lists := []connection.OptionList{{
		ConnectionID: "c0000beak", Field: "model", RowOptions: map[string]string{"reasoning": "low"},
		Operations: []string{"llm.chat"}, Revision: 3, PluginVersion: "1.2.0", Fields: map[string]string{"baseUrl": "https://finch.test/v1"},
		Options: []modelcall.Option{{Value: "finch-small", Title: "Finch small", Group: "Finch"}, {Value: "finch-large"}},
	}}

	newStorage(t, root, &reported).WriteOptionLists(finchPlugin, lists)

	read, isKept := newStorage(t, root, &reported).ReadOptionLists(finchPlugin)
	if !isKept || !reflect.DeepEqual(read, lists) || len(reported) != 0 {
		t.Errorf("read = %+v, kept = %v, reported = %v, want %+v", read, isKept, reported, lists)
	}
	if _, err := os.Stat(filepath.Join(root, "models", finchPlugin+".json")); err != nil {
		t.Errorf("the file of the plugin: %v", err)
	}
}

func TestUnreadableOptionListsAreReportedAndReadAsNone(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "models"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "models", finchPlugin+".json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var reported []error

	_, isKept := newStorage(t, root, &reported).ReadOptionLists(finchPlugin)

	if isKept || len(reported) != 1 {
		t.Errorf("kept = %v, reported = %v, want none kept and one report", isKept, reported)
	}
}

func TestAHostWithNoCacheRootKeepsNothing(t *testing.T) {
	var reported []error
	files := newStorage(t, "", &reported)

	files.WriteDetectedConnections(finchPlugin, []byte(`{"answers":{}}`))
	files.WriteOptionLists(finchPlugin, []connection.OptionList{{ConnectionID: "c0000beak", Field: "model"}})

	_, isConnectionsKept := files.ReadDetectedConnections(finchPlugin)
	_, isListsKept := files.ReadOptionLists(finchPlugin)
	if isConnectionsKept || isListsKept || len(reported) != 0 {
		t.Errorf("connections kept = %v, lists kept = %v, reported = %v", isConnectionsKept, isListsKept, reported)
	}
}
