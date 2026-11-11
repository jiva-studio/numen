// Package plugincopiesagree is the check `go/plugin-copies-agree`, run on the tree with the planted violations that prove it.
package plugincopiesagree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// getCopyDrifts defines the whole rule, so it is tested directly.
func TestWhatTheCopyRuleRejects(t *testing.T) {
	same := func() map[string][]byte { return map[string][]byte{"a.go": []byte("one"), "b.go": []byte("two")} }
	order := []string{"first", "second", "third"}
	for _, one := range []struct {
		name     string
		copies   map[string]map[string][]byte
		intended []string
		drifts   int
	}{
		{"three equal copies", map[string]map[string][]byte{"first": same(), "second": same(), "third": same()}, nil, 0},
		{"a file that differs in one copy", map[string]map[string][]byte{"first": same(), "second": same(), "third": {"a.go": []byte("one"), "b.go": []byte("2")}}, nil, 1},
		{
			"a file that differs in two copies",
			map[string]map[string][]byte{
				"first":  same(),
				"second": {"a.go": []byte("1"), "b.go": []byte("two")},
				"third":  {"a.go": []byte("1"), "b.go": []byte("two")},
			},
			nil,
			2,
		},
		{"a file a copy lacks", map[string]map[string][]byte{"first": same(), "second": {"a.go": []byte("one")}, "third": same()}, nil, 1},
		{"a file only a copy has", map[string]map[string][]byte{"first": same(), "second": same(), "third": {"a.go": []byte("one"), "b.go": []byte("two"), "c.go": nil}}, nil, 1},
		{"a file that is allowed to differ", map[string]map[string][]byte{"first": same(), "second": same(), "third": {"a.go": []byte("one"), "b.go": []byte("2")}}, []string{"b.go"}, 0},
		{"a file that is allowed to differ and is missing", map[string]map[string][]byte{"first": same(), "second": same(), "third": {"a.go": []byte("one")}}, []string{"b.go"}, 1},
	} {
		if got := getCopyDrifts(one.copies, order, one.intended); len(got) != one.drifts {
			t.Errorf("%s: drifts %v, want %d", one.name, got, one.drifts)
		}
	}
	if got := getCopyDrifts(map[string]map[string][]byte{"first": same()}, []string{"first"}, nil); len(got) != 1 {
		t.Errorf("one copy: drifts %v, want the refusal to compare nothing", got)
	}
}

// The copies of a folder agree file by file, apart from the files the rule names.
func TestTheCopiesOfAPluginFolderAgree(t *testing.T) {
	for _, set := range listCopySets() {
		copies := map[string]map[string][]byte{}
		for _, plugin := range set.Plugins {
			copies[plugin] = readFolder(t, filepath.Join(load.RepoRootDir, "plugins", plugin, filepath.FromSlash(set.Folder)))
		}
		for _, drift := range getCopyDrifts(copies, set.Plugins, set.Intended) {
			t.Errorf("%s: %s", set.Folder, drift)
		}
	}
}

func readFolder(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = content
	}
	return files
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
