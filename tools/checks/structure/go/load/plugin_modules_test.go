package load

import (
	"path/filepath"
	"testing"
)

func TestGetPluginModules_FindsThePlantedModules(t *testing.T) {
	root := WriteTreeFixture(t, map[string]string{
		"plugins/alpha/go.mod":  "module alpha\n",
		"plugins/beta/go.mod":   "module beta\n",
		"plugins/gamma/main.go": "package main\n",
		"other/delta/go.mod":    "module delta\n",
	})
	dirs, err := GetPluginModules(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "plugins", "alpha"), filepath.Join(root, "plugins", "beta")}
	if len(dirs) != len(want) || dirs[0] != want[0] || dirs[1] != want[1] {
		t.Errorf("found %v, want %v", dirs, want)
	}
}

func TestGetPluginModules_FailsWithoutAModule(t *testing.T) {
	if _, err := GetPluginModules(t.TempDir()); err == nil {
		t.Error("a tree without a plugin module passed")
	}
}
