package onnxruntime

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
	ort "github.com/getcharzp/onnxruntime_purego"
)

const (
	// allowSpinningKey is the session entry that lets idle intra-op threads
	// busy-wait between runs.
	allowSpinningKey = "session.intra_op.allow_spinning"

	// ortAPIVersion is the OrtApi table revision the slots below index into.
	ortAPIVersion = 23
)

// ortAPIConfig reaches the status and session-config slots of the OrtApi table.
type ortAPIConfig struct {
	_                     [2]uintptr
	GetErrorMessage       uintptr
	_                     [90]uintptr
	ReleaseStatus         uintptr
	_                     [36]uintptr
	AddSessionConfigEntry uintptr
}

// configEntryTable holds the runtime calls that set one session config entry.
type configEntryTable struct {
	addEntry      func(options uintptr, key, value *byte) uintptr
	getMessage    func(status uintptr) *byte
	releaseStatus func(status uintptr)
}

var (
	configEntryMu     sync.Mutex
	configEntryLoaded bool
	configEntryCalls  *configEntryTable
)

// loadConfigEntryTable resolves the config-entry calls of the library at path.
// It returns a nil table when the library does not carry them.
func loadConfigEntryTable(at string) (*configEntryTable, error) {
	handle, err := loadLibrary(at)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", at, err)
	}

	var getAPIBase func() *ortAPIBase
	purego.RegisterLibFunc(&getAPIBase, handle, "OrtGetApiBase")
	if getAPIBase == nil {
		return nil, nil
	}
	base := getAPIBase()
	if base == nil || base.GetAPI == 0 {
		return nil, nil
	}

	var getAPI func(uint32) *ortAPIConfig
	purego.RegisterFunc(&getAPI, base.GetAPI)
	api := getAPI(ortAPIVersion)
	if api == nil || api.AddSessionConfigEntry == 0 || api.GetErrorMessage == 0 || api.ReleaseStatus == 0 {
		return nil, nil
	}

	table := &configEntryTable{}
	purego.RegisterFunc(&table.addEntry, api.AddSessionConfigEntry)
	purego.RegisterFunc(&table.getMessage, api.GetErrorMessage)
	purego.RegisterFunc(&table.releaseStatus, api.ReleaseStatus)
	return table, nil
}

// getConfigEntryTable returns the config-entry calls of the loaded runtime,
// resolving them on first use.
func getConfigEntryTable() (*configEntryTable, error) {
	_, at := runtimeState.GetLoaded()
	if at == "" {
		return nil, fmt.Errorf("onnx runtime not loaded")
	}

	configEntryMu.Lock()
	defer configEntryMu.Unlock()
	if configEntryLoaded {
		return configEntryCalls, nil
	}
	table, err := loadConfigEntryTable(at)
	if err != nil {
		return nil, err
	}
	configEntryCalls, configEntryLoaded = table, true
	return table, nil
}

// addConfigEntry sets one session config entry through the table. A nil table
// leaves the options untouched.
func addConfigEntry(table *configEntryTable, options uintptr, key, value string) error {
	if table == nil || table.addEntry == nil {
		return nil
	}
	keyPtr, err := syscall.BytePtrFromString(key)
	if err != nil {
		return fmt.Errorf("session config key %q: %w", key, err)
	}
	valuePtr, err := syscall.BytePtrFromString(value)
	if err != nil {
		return fmt.Errorf("session config value for %q: %w", key, err)
	}

	status := table.addEntry(options, keyPtr, valuePtr)
	if status == 0 {
		return nil
	}
	message := "unknown error"
	if table.getMessage != nil {
		if text := readCString(table.getMessage(status)); text != "" {
			message = text
		}
	}
	if table.releaseStatus != nil {
		table.releaseStatus(status)
	}
	return fmt.Errorf("set session config %s=%s: %s", key, value, message)
}

// readCString copies a NUL-terminated string out of runtime memory.
func readCString(p *byte) string {
	if p == nil {
		return ""
	}
	var out []byte
	for ; *p != 0; p = (*byte)(unsafe.Add(unsafe.Pointer(p), 1)) {
		out = append(out, *p)
	}
	return string(out)
}

// disableSpinning stops the session's idle intra-op threads from busy-waiting.
// Idle threads do not spin because the application shares the CPU with its
// interface.
func disableSpinning(opts *ort.SessionOptions) error {
	optHandle := *(*uintptr)(unsafe.Pointer(opts))
	if optHandle == 0 {
		return fmt.Errorf("invalid session options handle")
	}
	table, err := getConfigEntryTable()
	if err != nil {
		return fmt.Errorf("resolve session config calls: %w", err)
	}
	return addConfigEntry(table, optHandle, allowSpinningKey, "0")
}
