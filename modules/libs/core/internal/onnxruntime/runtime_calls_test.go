package onnxruntime

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// useRuntimeAt leaves the process with the runtime at the given place, and no
// call into it resolved.
func useRuntimeAt(t *testing.T, at string) {
	t.Helper()
	runtimeState.mu.Lock()
	previous := runtimeState.at
	runtimeState.at = at
	runtimeState.mu.Unlock()

	optLevelMu.Lock()
	level := setSessionGraphOptimizationLevel
	setSessionGraphOptimizationLevel = nil
	optLevelMu.Unlock()

	configEntryMu.Lock()
	loaded, calls := configEntryLoaded, configEntryCalls
	configEntryLoaded, configEntryCalls = false, nil
	configEntryMu.Unlock()

	t.Cleanup(func() {
		runtimeState.mu.Lock()
		runtimeState.at = previous
		runtimeState.mu.Unlock()
		optLevelMu.Lock()
		setSessionGraphOptimizationLevel = level
		optLevelMu.Unlock()
		configEntryMu.Lock()
		configEntryLoaded, configEntryCalls = loaded, calls
		configEntryMu.Unlock()
	})
}

func useOptimizationCall(fn func(options uintptr, level uint32) uintptr) {
	optLevelMu.Lock()
	setSessionGraphOptimizationLevel = fn
	optLevelMu.Unlock()
}

// optionsHolding is session options whose handle is the given one.
func optionsHolding(handle uintptr) (*ort.SessionOptions, *uintptr) {
	holder := new(uintptr)
	*holder = handle
	return (*ort.SessionOptions)(unsafe.Pointer(holder)), holder
}

func TestSetGraphOptimizationLevelCallsTheRuntimeWithTheHandleAndLevel(t *testing.T) {
	useRuntimeAt(t, "")
	var gotHandle uintptr
	var gotLevel uint32
	useOptimizationCall(func(options uintptr, level uint32) uintptr {
		gotHandle, gotLevel = options, level
		return 0
	})
	opts, holder := optionsHolding(42)

	if err := setGraphOptimizationLevel(opts, 99); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(holder)
	if gotHandle != 42 || gotLevel != 99 {
		t.Errorf("runtime called with handle %d level %d", gotHandle, gotLevel)
	}
}

func TestSetGraphOptimizationLevelReportsTheStatus(t *testing.T) {
	useRuntimeAt(t, "")
	useOptimizationCall(func(uintptr, uint32) uintptr { return 6 })
	opts, holder := optionsHolding(42)

	err := setGraphOptimizationLevel(opts, 99)
	runtime.KeepAlive(holder)
	if err == nil || !strings.Contains(err.Error(), "6") {
		t.Errorf("error = %v, want the status 6", err)
	}
}

func TestSetGraphOptimizationLevelRefusesAnEmptyHandle(t *testing.T) {
	useRuntimeAt(t, "")
	called := false
	useOptimizationCall(func(uintptr, uint32) uintptr { called = true; return 0 })
	opts, holder := optionsHolding(0)

	err := setGraphOptimizationLevel(opts, 99)
	runtime.KeepAlive(holder)
	if err == nil || !strings.Contains(err.Error(), "invalid session options handle") {
		t.Errorf("error = %v, want the handle refused", err)
	}
	if called {
		t.Error("the runtime was called with an empty handle")
	}
}

func TestSetGraphOptimizationLevelNeedsALoadedRuntime(t *testing.T) {
	useRuntimeAt(t, "")
	opts, holder := optionsHolding(42)

	err := setGraphOptimizationLevel(opts, 99)
	runtime.KeepAlive(holder)
	if err == nil || !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("error = %v, want the runtime reported as not loaded", err)
	}
}

func TestSetGraphOptimizationLevelReportsALibraryThatDoesNotOpen(t *testing.T) {
	useRuntimeAt(t, filepath.Join(t.TempDir(), "libonnxruntime-missing"))
	opts, holder := optionsHolding(42)

	err := setGraphOptimizationLevel(opts, 99)
	runtime.KeepAlive(holder)
	if err == nil || strings.Contains(err.Error(), "not loaded") {
		t.Errorf("error = %v, want the library's own failure", err)
	}
}

func TestInitGraphOptimizationKeepsTheCallItAlreadyHas(t *testing.T) {
	useRuntimeAt(t, "")
	useOptimizationCall(func(uintptr, uint32) uintptr { return 0 })

	if err := initGraphOptimization(""); err != nil {
		t.Errorf("error = %v, want none", err)
	}
}

func TestInitGraphOptimizationNeedsAPlaceToLoadFrom(t *testing.T) {
	useRuntimeAt(t, "")

	err := initGraphOptimization("")
	if err == nil || !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("error = %v, want the runtime reported as not loaded", err)
	}
	if err := initGraphOptimization(filepath.Join(t.TempDir(), "libonnxruntime-missing")); err == nil {
		t.Error("a library that does not exist was loaded")
	}
}

func TestDisableSpinningRefusesAnEmptyHandle(t *testing.T) {
	useRuntimeAt(t, "")
	opts, holder := optionsHolding(0)

	err := disableSpinning(opts)
	runtime.KeepAlive(holder)
	if err == nil || !strings.Contains(err.Error(), "invalid session options handle") {
		t.Errorf("error = %v, want the handle refused", err)
	}
}

func TestDisableSpinningKeepsTheCauseWhenTheRuntimeIsNotLoaded(t *testing.T) {
	useRuntimeAt(t, "")
	opts, holder := optionsHolding(42)

	err := disableSpinning(opts)
	runtime.KeepAlive(holder)
	if err == nil || !strings.Contains(err.Error(), "resolve session config calls") || errors.Unwrap(err) == nil {
		t.Errorf("error = %v, want the cause wrapped", err)
	}
}

func TestGetConfigEntryTableNeedsALoadedRuntime(t *testing.T) {
	useRuntimeAt(t, "")

	table, err := getConfigEntryTable()
	if err == nil || table != nil {
		t.Errorf("getConfigEntryTable = %v, %v; want an error", table, err)
	}
}

func TestGetConfigEntryTableKeepsWhatItResolved(t *testing.T) {
	useRuntimeAt(t, filepath.Join(t.TempDir(), "libonnxruntime-missing"))
	kept := &configEntryTable{}
	configEntryMu.Lock()
	configEntryLoaded, configEntryCalls = true, kept
	configEntryMu.Unlock()

	table, err := getConfigEntryTable()
	if err != nil || table != kept {
		t.Errorf("getConfigEntryTable = %v, %v; want the table already resolved", table, err)
	}
}

func TestGetConfigEntryTableReportsALibraryThatDoesNotOpen(t *testing.T) {
	useRuntimeAt(t, filepath.Join(t.TempDir(), "libonnxruntime-missing"))

	table, err := getConfigEntryTable()
	if err == nil || table != nil {
		t.Errorf("getConfigEntryTable = %v, %v; want an error", table, err)
	}
	if configEntryLoaded {
		t.Error("a failed resolve was kept")
	}
}

func TestLoadConfigEntryTableKeepsTheCauseOfALibraryThatDoesNotOpen(t *testing.T) {
	table, err := loadConfigEntryTable(filepath.Join(t.TempDir(), "libonnxruntime-missing"))
	if err == nil || table != nil || errors.Unwrap(err) == nil {
		t.Errorf("loadConfigEntryTable = %v, %v; want an error wrapping the cause", table, err)
	}
}

func TestTheRuntimeNameIsThePlatformsLibrary(t *testing.T) {
	want := map[string]string{
		"darwin":  "libonnxruntime.dylib",
		"windows": "onnxruntime.dll",
	}[runtime.GOOS]
	if want == "" {
		want = "libonnxruntime.so"
	}
	if got := runtimeName(); got != want {
		t.Errorf("runtimeName = %s, want %s", got, want)
	}
	found := false
	for _, name := range runtimeNames {
		found = found || name == runtimeName()
	}
	if !found {
		t.Errorf("%s is not among the names looked for", runtimeName())
	}
}

func TestAPlatformIsPublishedWhenItsRuntimeIsKnown(t *testing.T) {
	published := map[string]bool{
		"linux/amd64": true, "linux/arm64": true, "darwin/arm64": true,
		"darwin/amd64": true, "windows/amd64": true,
	}[runtime.GOOS+"/"+runtime.GOARCH]
	if IsPublished() != published {
		t.Errorf("IsPublished = %v on %s/%s", IsPublished(), runtime.GOOS, runtime.GOARCH)
	}
	found, err := findRelease(getSettings())
	if published && (err != nil || found != releases[runtime.GOOS+"/"+runtime.GOARCH]) {
		t.Errorf("findRelease = %+v, %v", found, err)
	}
	if !published && (err == nil || !strings.Contains(err.Error(), getSettings().Section)) {
		t.Errorf("findRelease on an unpublished platform = %v", err)
	}
}
