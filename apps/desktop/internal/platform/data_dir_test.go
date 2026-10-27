package platform

import (
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
