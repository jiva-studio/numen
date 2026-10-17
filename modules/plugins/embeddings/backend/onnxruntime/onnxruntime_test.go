package onnxruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRefusesWhenTheHostGaveNoLibrary(t *testing.T) {
	if _, _, err := Open(context.Background(), Settings{}); err == nil || !strings.Contains(err.Error(), "the host names the library") {
		t.Fatalf("Open() = %v, want the host named", err)
	}
}

func TestOpenRefusesALibraryThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if _, _, err := Open(context.Background(), Settings{Runtime: missing}); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want %s named", err, missing)
	}
}

func TestALibraryIsAvailableWhereTheFileIs(t *testing.T) {
	if IsAvailable(Settings{}) {
		t.Error("no library is available when none is named")
	}
	at := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if IsAvailable(Settings{Runtime: at}) {
		t.Error("a missing file is available")
	}
	if err := os.WriteFile(at, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsAvailable(Settings{Runtime: at}) {
		t.Error("a file that is there is not available")
	}
}
