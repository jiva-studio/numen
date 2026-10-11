package onnxruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

// recordTable returns a table whose add call records what it was given and
// answers with the status.
func recordTable(status uintptr, key, value *string, options *uintptr) *configEntryTable {
	return &configEntryTable{
		addEntry: func(o uintptr, k, v *byte) uintptr {
			*options, *key, *value = o, readCString(k), readCString(v)
			return status
		},
		getMessage: func(uintptr) *byte {
			message := []byte("boom\x00")
			return &message[0]
		},
		releaseStatus: func(uintptr) {},
	}
}

func TestAddConfigEntryPassesKeyAndValueThroughTheTable(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(0, &key, &value, &options)

	if err := addConfigEntry(table, 42, allowSpinningKey, "0"); err != nil {
		t.Fatal(err)
	}
	if options != 42 || key != "session.intra_op.allow_spinning" || value != "0" {
		t.Errorf("table got options=%d key=%q value=%q", options, key, value)
	}
}

func TestAddConfigEntryReturnsTheStatusMessage(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(7, &key, &value, &options)

	err := addConfigEntry(table, 1, allowSpinningKey, "0")
	if err == nil || !strings.Contains(err.Error(), "boom") || !strings.Contains(err.Error(), allowSpinningKey) {
		t.Errorf("error = %v, want the key and the runtime message", err)
	}
}

func TestAddConfigEntryIsANoOpWithoutTheEntryPoint(t *testing.T) {
	if err := addConfigEntry(nil, 1, allowSpinningKey, "0"); err != nil {
		t.Errorf("nil table: %v", err)
	}
	if err := addConfigEntry(&configEntryTable{}, 1, allowSpinningKey, "0"); err != nil {
		t.Errorf("empty table: %v", err)
	}
}

func TestAddConfigEntryRejectsAnEmbeddedNUL(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(0, &key, &value, &options)
	if err := addConfigEntry(table, 1, "a\x00b", "0"); err == nil {
		t.Error("expected an error for a key holding NUL")
	}
}

// The slot offsets are read from the library on this machine: a wrong offset
// either crashes or fails the call.
func TestConfigEntryTableMatchesTheLoadedRuntime(t *testing.T) {
	engine, at, err := Open(t.Context(), getSettings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	table, err := loadConfigEntryTable(at)
	if err != nil {
		t.Fatal(err)
	}
	if table == nil {
		t.Skip("this runtime has no session config entry point")
	}
	opts, err := engine.NewSessionOptions()
	if err != nil {
		t.Fatal(err)
	}
	defer opts.Destroy()
	handle := *(*uintptr)(unsafe.Pointer(opts))

	if err := addConfigEntry(table, handle, allowSpinningKey, "0"); err != nil {
		t.Errorf("valid entry refused: %v", err)
	}
	err = addConfigEntry(table, handle, "", "0")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("empty key = %v, want the runtime's refusal", err)
	}
}

// findSmallestModel returns the smallest model in the user's model cache.
func findSmallestModel(t *testing.T) string {
	t.Helper()
	dir, err := os.UserCacheDir()
	if err != nil {
		t.Skip(err)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "numen", "models", "*.onnx"))
	smallest, size := "", int64(0)
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && (smallest == "" || info.Size() < size) {
			smallest, size = path, info.Size()
		}
	}
	if smallest == "" {
		t.Skip("no model in the cache")
	}
	return smallest
}

func TestNewSessionOptionsBuildForEveryProvider(t *testing.T) {
	engine, _, err := Open(t.Context(), getSettings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	for _, provider := range []Provider{ProviderCPU, ProviderAuto, ProviderCUDA, ProviderCoreML, ProviderDirectML, ProviderROCm} {
		t.Run(string(provider), func(t *testing.T) {
			opts, _, err := NewSessionOptions(engine, SessionSettings{Provider: provider, Threads: 1})
			if err != nil {
				t.Skipf("provider unavailable here: %v", err)
			}
			opts.Destroy()
		})
	}
}

func TestSessionOpensWithSpinningDisabled(t *testing.T) {
	engine, _, err := Open(t.Context(), getSettings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	opts, _, err := NewSessionOptions(engine, SessionSettings{Provider: ProviderCPU, Threads: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer opts.Destroy()

	session, err := engine.NewSession(findSmallestModel(t), opts)
	if err != nil {
		t.Fatal(err)
	}
	session.Destroy()
}
