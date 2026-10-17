package onnxruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func TestOpenRefusesWhenNoLibraryIsNamed(t *testing.T) {
	t.Setenv(onnx.EnvLibrary, "")
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

func TestTheLibraryTheHostGaveIsOpenedWhenNoSettingNamesOne(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libhost.so")
	t.Setenv(onnx.EnvLibrary, missing)
	if _, _, err := Open(context.Background(), Settings{}); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want the host's %s tried", err, missing)
	}
	if got := (Settings{Runtime: "mine.so"}).library(); got != "mine.so" {
		t.Errorf("library() = %q, want the setting to win", got)
	}
}

func TestALibraryIsAvailableWhereTheFileIs(t *testing.T) {
	t.Setenv(onnx.EnvLibrary, "")
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
	t.Setenv(onnx.EnvLibrary, at)
	if !IsAvailable(Settings{}) {
		t.Error("the file the host named is not available")
	}
}
