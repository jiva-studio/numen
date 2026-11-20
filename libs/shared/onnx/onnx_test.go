package onnx

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestGetLibraryAndGetProviderAreWhatTheHostSet(t *testing.T) {
	t.Setenv(EnvLibrary, "/models/libonnxruntime.so")
	t.Setenv(EnvProvider, "coreml")
	if got := GetLibrary(); got != "/models/libonnxruntime.so" {
		t.Errorf("GetLibrary() = %q", got)
	}
	if got := GetProvider(""); got != "coreml" {
		t.Errorf("GetProvider() = %q", got)
	}
}

func TestGetProvider_DefaultsToAutoWithoutHostRuntime(t *testing.T) {
	t.Setenv(EnvLibrary, "")
	t.Setenv(EnvProvider, "")
	if GetLibrary() != "" || GetProvider("") != DefaultProvider {
		t.Errorf("GetLibrary() = %q, GetProvider() = %q", GetLibrary(), GetProvider(""))
	}
}

func TestResolveLibraryIsTheSettingThenTheFolderThenTheHost(t *testing.T) {
	t.Setenv(EnvLibrary, "/host/libonnxruntime.so")
	dir := t.TempDir()
	targetPath := filepath.Join(dir, GetLibraryName())
	if err := os.WriteFile(targetPath, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, one := range []struct{ configured, dir, want string }{
		{"mine.so", dir, "mine.so"},
		{"", dir, targetPath},
		{"", t.TempDir(), "/host/libonnxruntime.so"},
		{"", "", "/host/libonnxruntime.so"},
	} {
		if got := ResolveLibrary(one.configured, one.dir); got != one.want {
			t.Errorf("ResolveLibrary(%q, %q) = %q, want %q", one.configured, one.dir, got, one.want)
		}
	}
}

func TestGetSessionProviderIsTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for _, one := range []struct {
		name string
		want Provider
	}{{"", ProviderCPU}, {"auto", ProviderCPU}, {" AUTO ", ProviderCPU}, {"cuda", ProviderCUDA}} {
		t.Setenv(EnvProvider, one.name)
		if got := GetSessionProvider(""); got != one.want {
			t.Errorf("with %q, GetSessionProvider() = %q, want %q", one.name, got, one.want)
		}
	}
}

func TestResolveLibraryReadsTheHostVariableAtEachCall(t *testing.T) {
	t.Setenv(EnvLibrary, "")
	if got := ResolveLibrary("", ""); got != "" {
		t.Fatalf("ResolveLibrary() = %q before the host named a library", got)
	}

	t.Setenv(EnvLibrary, "/host/libonnxruntime.so")

	if got := ResolveLibrary("", ""); got != "/host/libonnxruntime.so" {
		t.Errorf("ResolveLibrary() = %q after the host named a library, want it", got)
	}
}

func TestSessionProviderUsesTheSharedSelectionUnlessOverridden(t *testing.T) {
	t.Setenv(EnvProvider, "")
	if got := GetSessionProvider("cuda"); got != ProviderCUDA {
		t.Fatalf("persisted provider = %v", got)
	}
	if got := GetProvider("auto"); got != ProviderAuto {
		t.Fatalf("embeddings auto = %v", got)
	}
	if got := GetSessionProvider("auto"); got != ProviderCPU {
		t.Fatalf("default session = %v", got)
	}
	t.Setenv(EnvProvider, "cpu")
	if got := GetSessionProvider("cuda"); got != ProviderCPU {
		t.Fatalf("override = %v", got)
	}
}

func TestSessionThreadsRespectTheHostLimit(t *testing.T) {
	for _, one := range []struct{ requested, limit, want int }{{8, 2, 2}, {2, 8, 2}, {4, 0, 4}} {
		if got := LimitThreads(one.requested, one.limit); got != one.want {
			t.Fatalf("LimitThreads(%d,%d)=%d", one.requested, one.limit, got)
		}
	}
}

func TestSessionRejectsAnOversizeThreadCountBeforeAllocatingOptions(t *testing.T) {
	if strconv.IntSize <= 32 {
		t.Skip("int fits in the native thread count")
	}
	count := int64(math.MaxInt32) + 1
	_, _, err := NewLibrary().NewSessionOptions(nil, SessionSettings{Threads: int(count)})
	if err == nil {
		t.Fatal("accepted a thread count that overflows int32")
	}
}
