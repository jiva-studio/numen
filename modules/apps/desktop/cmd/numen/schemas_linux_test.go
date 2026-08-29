package main

import (
	"os"
	"path/filepath"
	"testing"
)

// schemasIn writes the file a folder of compiled settings is recognised by.
func schemasIn(t *testing.T, dir string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gschemas.compiled"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSettingsAreFoundInADataDirectory(t *testing.T) {
	data := t.TempDir()
	schemasIn(t, filepath.Join(data, "glib-2.0", "schemas"))
	t.Setenv("GSETTINGS_SCHEMA_DIR", "")
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty")+":"+data)

	if !settled() {
		t.Error("the settings were not found where they are")
	}
}

func TestSettingsAreFoundWhereTheEnvironmentNamesThem(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("GSETTINGS_SCHEMA_DIR", schemasIn(t, filepath.Join(t.TempDir(), "held")))

	if !settled() {
		t.Error("the settings were not found where the environment names them")
	}
}

func TestAMachineWithNoSettingsIsSaidToHaveNone(t *testing.T) {
	t.Setenv("GSETTINGS_SCHEMA_DIR", "")
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty"))

	if settled() {
		t.Error("a machine holding no settings was said to hold them")
	}
}

func TestTheToolkitIsLookedThroughForItsSettings(t *testing.T) {
	// The folder the toolkit was loaded from carries its own settings, either
	// where every package puts them or filed under the package they came from.
	for _, at := range []string{
		filepath.Join("share", "glib-2.0", "schemas"),
		filepath.Join("share", "gsettings-schemas", "gtk4-4.22.4", "glib-2.0", "schemas"),
	} {
		prefix := t.TempDir()
		want := schemasIn(t, filepath.Join(prefix, at))

		if got := schemasOf(prefix); got != want {
			t.Errorf("under %s the settings were %q, want %q", at, got, want)
		}
	}
}

func TestAToolkitCarryingNoSettingsAnswersWithNone(t *testing.T) {
	if got := schemasOf(t.TempDir()); got != "" {
		t.Errorf("settings were claimed at %q", got)
	}
	if got := schemasOf(""); got != "" {
		t.Errorf("settings were claimed for a toolkit that was not found: %q", got)
	}
}

func TestTheSettingsTheEnvironmentNamesAreKept(t *testing.T) {
	// A machine that named a folder of its own is added to, and what it named
	// stays reachable.
	theirs := schemasIn(t, filepath.Join(t.TempDir(), "theirs"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("GSETTINGS_SCHEMA_DIR", theirs)

	findSchemas()

	if got := os.Getenv("GSETTINGS_SCHEMA_DIR"); got != theirs {
		t.Errorf("the folder the machine named became %q", got)
	}
}
