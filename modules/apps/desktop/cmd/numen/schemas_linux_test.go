package main

import (
	"os"
	"path/filepath"
	"testing"
)

// schemasAt writes the file a data directory carries its compiled settings in,
// and answers with the directory to name as a data directory.
func schemasAt(t *testing.T) string {
	t.Helper()

	data := t.TempDir()
	dir := filepath.Join(data, "glib-2.0", "schemas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gschemas.compiled"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSettingsAreFoundInADataDirectory(t *testing.T) {
	t.Setenv("GSETTINGS_SCHEMA_DIR", "")
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty")+":"+schemasAt(t))

	if !settled() {
		t.Error("the settings were not found where they are")
	}
}

func TestSettingsAreFoundWhereTheEnvironmentNamesThem(t *testing.T) {
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty"))
	t.Setenv("GSETTINGS_SCHEMA_DIR", filepath.Join(schemasAt(t), "glib-2.0", "schemas"))

	if !settled() {
		t.Error("the settings were not found where the environment names them")
	}
}

func TestAMachineWithNoSettingsIsSaidToHaveNone(t *testing.T) {
	// A picker asked for here ends the process, so this is the answer the
	// window turns into words.
	t.Setenv("GSETTINGS_SCHEMA_DIR", "")
	t.Setenv("XDG_DATA_DIRS", filepath.Join(t.TempDir(), "empty"))

	if settled() {
		t.Error("a machine holding no settings was said to hold them")
	}
}
