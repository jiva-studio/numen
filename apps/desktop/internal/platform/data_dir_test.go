package platform

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetDataDirFollowsEachOperatingSystem(t *testing.T) {
	home := filepath.Join(string(filepath.Separator), "home", "ada")
	getHome := func() (string, error) { return home, nil }
	cases := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{"linux default", "linux", nil, filepath.Join(home, ".local", "share")},
		{"linux XDG_DATA_HOME", "linux", map[string]string{"XDG_DATA_HOME": "/data/ada"}, "/data/ada"},
		{"linux relative XDG_DATA_HOME", "linux", map[string]string{"XDG_DATA_HOME": "data"}, filepath.Join(home, ".local", "share")},
		{"macOS", "darwin", nil, filepath.Join(home, "Library", "Application Support")},
		{"Windows", "windows", map[string]string{"AppData": `C:\Users\ada\AppData\Roaming`}, `C:\Users\ada\AppData\Roaming`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := getDataDir(c.goos, func(key string) string { return c.env[key] }, getHome)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestGetDataDirRejectsWindowsWithoutAppData(t *testing.T) {
	if _, err := getDataDir("windows", func(string) string { return "" }, nil); err == nil {
		t.Fatal("a Windows machine without %AppData% gave a data folder")
	}
}

func TestGetConfigNamesTheIndexInTheDataFolder(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_DATA_HOME is read on Linux")
	}
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	if got, want := GetConfig().IndexPath, filepath.Join(data, "numen", "index.db"); got != want {
		t.Errorf("index path %q, want %q", got, want)
	}
}

func TestStorageRootsAreInsideTheCacheAndDataFolders(t *testing.T) {
	cache := func() (string, error) { return "/cache", nil }
	home := func() (string, error) { return "/home/user", nil }
	env := func(string) string { return "" }

	cacheRoot, dataRoot := getStorageRoots("linux", env, cache, home)

	if want := "/cache/numen/plugin-storage"; cacheRoot != want {
		t.Errorf("cache root %q, want %q", cacheRoot, want)
	}
	if want := "/home/user/.local/share/numen/plugin-storage"; dataRoot != want {
		t.Errorf("data root %q, want %q", dataRoot, want)
	}
}

func TestAFolderTheMachineDoesNotProvideHasNoRoot(t *testing.T) {
	fail := func() (string, error) { return "/ignored", errors.New("no folder") }
	empty := func() (string, error) { return "", nil }
	env := func(string) string { return "" }

	if _, err := getDataDir("darwin", env, fail); err == nil {
		t.Error("expected error on darwin when getHome fails")
	}
	if _, err := getDataDir("linux", env, fail); err == nil {
		t.Error("expected error on linux when getHome fails")
	}

	cacheRoot, dataRoot := getStorageRoots("linux", env, fail, fail)
	if cacheRoot != "" || dataRoot != "" {
		t.Errorf("roots %q and %q, want both empty", cacheRoot, dataRoot)
	}

	cacheRoot, _ = getStorageRoots("linux", env, empty, fail)
	if cacheRoot != "" {
		t.Errorf("cache root %q, want empty", cacheRoot)
	}

	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "")
	if _, err := GetIndexPath(); err == nil {
		t.Error("expected GetIndexPath error when home is unset")
	}
	if got := GetConfig().IndexPath; got != "" {
		t.Errorf("IndexPath = %q, want empty when home is unset", got)
	}
}
