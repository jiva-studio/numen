package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGetOpenFailure(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "corrupt.db")
	if err := os.WriteFile(corrupt, []byte("PK\x03\x04 not a database"), 0o644); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(dir, "folder.db")
	if err := os.Mkdir(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing", "index.db")

	tests := map[string]struct {
		path string
		want OpenFailure
	}{
		"a file that is not a database": {corrupt, NotAnIndex},
		"a path that is a folder":       {folder, PathIsFolder},
		"a folder that does not exist":  {missing, CannotCreate},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			db, err := Open(t.Context(), test.path)
			if err == nil {
				if closeErr := db.Close(); closeErr != nil {
					t.Error(closeErr)
				}
				t.Fatalf("%s opened as an index", test.path)
			}
			got, ok := GetOpenFailure(test.path, err)
			if !ok || got != test.want {
				t.Errorf("GetOpenFailure = %v, %v, want %v, true", got, ok, test.want)
			}
		})
	}

	if _, ok := GetOpenFailure(dir, errors.New("no result code")); ok {
		t.Error("an error without a result code was classified")
	}
}
