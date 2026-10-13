package onnxruntime

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

func TestAddConfigEntryNamesTheRejectedKeyAndKeepsTheCause(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(0, &key, &value, &options)

	err := addConfigEntry(table, 1, "a\x00b", "0")
	if !errors.Is(err, syscall.EINVAL) {
		t.Errorf("error = %v, want one wrapping %v", err, syscall.EINVAL)
	}
}

func TestAddConfigEntryNamesTheRejectedValueAndKeepsTheCause(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(0, &key, &value, &options)

	err := addConfigEntry(table, 1, allowSpinningKey, "0\x001")
	if !errors.Is(err, syscall.EINVAL) {
		t.Fatalf("error = %v, want one wrapping %v", err, syscall.EINVAL)
	}
	if !strings.Contains(err.Error(), allowSpinningKey) {
		t.Errorf("error = %v, want the key named", err)
	}
	if key != "" {
		t.Errorf("the runtime was called with key %q though the value was refused", key)
	}
}

func TestAddConfigEntryReleasesTheStatusItReports(t *testing.T) {
	var key, value string
	var options uintptr
	table := recordTable(7, &key, &value, &options)
	var released []uintptr
	table.releaseStatus = func(status uintptr) { released = append(released, status) }

	if err := addConfigEntry(table, 1, allowSpinningKey, "0"); err == nil {
		t.Fatal("expected the status to be reported")
	}
	if len(released) != 1 || released[0] != 7 {
		t.Errorf("released = %v, want the status once", released)
	}
}

func TestAddConfigEntryReportsAStatusWithoutReleaseOrMessageCalls(t *testing.T) {
	table := &configEntryTable{addEntry: func(uintptr, *byte, *byte) uintptr { return 3 }}

	err := addConfigEntry(table, 1, allowSpinningKey, "0")
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("error = %v, want the unknown error", err)
	}
}

func TestAddConfigEntryFallsBackWhenTheMessageIsEmpty(t *testing.T) {
	table := &configEntryTable{
		addEntry:   func(uintptr, *byte, *byte) uintptr { return 3 },
		getMessage: func(uintptr) *byte { return nil },
	}

	err := addConfigEntry(table, 1, allowSpinningKey, "0")
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("error = %v, want the unknown error", err)
	}
}

func TestReadCStringCopiesUpToTheTerminator(t *testing.T) {
	text := []byte("abc\x00def\x00")
	if got := readCString(&text[0]); got != "abc" {
		t.Errorf("readCString = %q, want abc", got)
	}
	if got := readCString(nil); got != "" {
		t.Errorf("readCString(nil) = %q, want empty", got)
	}
}

// The runtime is asked for the OrtApi table revision of the release this build
// pins, and an older release answers that it does not have it.
func TestTheApiRevisionIsTheOneThePinnedRuntimeCarries(t *testing.T) {
	minor := strings.Split(runtimeVersion, ".")[1]
	if strconv.Itoa(ortAPIVersion) != minor {
		t.Errorf("API revision %d, runtime %s", ortAPIVersion, runtimeVersion)
	}
}

// The slots are counted from the start of the OrtApi table.
func TestTheOrtApiSlotsSitWhereTheTableKeepsThem(t *testing.T) {
	word := unsafe.Sizeof(uintptr(0))
	var config ortAPIConfig
	var optimization ortAPI
	for name, got := range map[string][2]uintptr{
		"GetErrorMessage":                  {unsafe.Offsetof(config.GetErrorMessage), 2 * word},
		"ReleaseStatus":                    {unsafe.Offsetof(config.ReleaseStatus), 93 * word},
		"AddSessionConfigEntry":            {unsafe.Offsetof(config.AddSessionConfigEntry), 130 * word},
		"SetSessionGraphOptimizationLevel": {unsafe.Offsetof(optimization.SetSessionGraphOptimizationLevel), 23 * word},
	} {
		if got[0] != got[1] {
			t.Errorf("%s at %d, want %d", name, got[0], got[1])
		}
	}
}
