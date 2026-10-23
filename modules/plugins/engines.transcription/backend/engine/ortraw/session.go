package ortraw

import (
	"unsafe"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// GetSessionHandle is the runtime's handle of a session, or nothing when there
// is none. A Session begins with its runtime handle.
func GetSessionHandle(session *ort.Session) uintptr {
	if session == nil {
		return 0
	}
	return *(*uintptr)(unsafe.Pointer(session))
}

// GetValueHandle is the runtime's handle of a value. A Value begins with its
// runtime handle.
func GetValueHandle(value *ort.Value) uintptr {
	return *(*uintptr)(unsafe.Pointer(value))
}

// GetNames are the names a graph answered with.
func GetNames(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	return out
}
