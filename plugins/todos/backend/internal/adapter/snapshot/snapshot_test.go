package snapshot

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

func sample() ports.Snapshot {
	return ports.Snapshot{
		Cursor: host.FileChangeCursor{FeedID: "feed-1", Seq: 4},
		Notes: map[string]ports.SavedNote{
			"a.md": {Hash: []byte{1, 2, 3}, Tasks: []todo.Task{{Path: "a.md", Line: 1, Text: "one", Projects: []string{"home"}}}},
		},
	}
}

func TestASavedSnapshotIsLoadedByAnotherStoreOfTheSameDirectory(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	if err := New("").Save(sample()); err != nil {
		t.Fatal(err)
	}

	got, isFound := New("").Load(t.Context())

	if !isFound || !reflect.DeepEqual(got, sample()) {
		t.Errorf("Load = %+v %v, want %+v", got, isFound, sample())
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "index.json" {
		t.Errorf("files in the cache directory = %v, want only index.json", entries)
	}
}

func TestAMissingOrCorruptFileIsNoSnapshot(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	if _, isFound := New("").Load(t.Context()); isFound {
		t.Error("a missing file was loaded")
	}

	if err := os.WriteFile(filepath.Join(cacheDir, "index.json"), []byte(`{"snapshot": {"value": [1`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, isFound := New("").Load(t.Context()); isFound {
		t.Error("a corrupt file was loaded")
	}
}

func TestAFileOfAnotherShapeIsNoSnapshot(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	if err := os.WriteFile(filepath.Join(cacheDir, "index.json"), []byte(`{"snapshot": {"value": "old"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, isFound := New("").Load(t.Context()); isFound {
		t.Error("a file of another shape was loaded")
	}
}
