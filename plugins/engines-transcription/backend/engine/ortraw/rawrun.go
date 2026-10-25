package ortraw

import (
	"fmt"
	"runtime"
	"unsafe"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// MakeCNames are names as the NUL-terminated C strings the runtime reads.
func MakeCNames(names ...string) [][]byte {
	out := make([][]byte, len(names))
	for i, name := range names {
		out[i] = append([]byte(name), 0)
	}
	return out
}

// Calls are the runtime entry points a Graph uses.
type Calls struct {
	Run     func(session, options, inNames, inValues, inCount, outNames, outCount, outValues uintptr) uintptr
	GetData func(value, out uintptr) uintptr
	Release func(value uintptr)
	Check   func(status uintptr) error
}

// GetCalls are the runtime entry points, or false when the runtime has not
// registered them.
func GetCalls() (Calls, bool) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	calls := Calls{Run: ortRun, GetData: ortGetTensorMutableData, Release: ortReleaseValue, Check: checkStatus}
	return calls, calls.Run != nil && calls.GetData != nil && calls.Release != nil
}

// A Graph is one graph run over and over on the same input tensors. It owns
// its output values: the first run lets the runtime create them, every later
// run writes into them, and close releases them.
type Graph struct {
	calls   Calls
	session uintptr

	inNames  []unsafe.Pointer
	outNames []unsafe.Pointer
	inputs   []uintptr
	outputs  []uintptr
	data     unsafe.Pointer
}

// NewGraph is a Graph of session over inputs, which are given in the order of
// inNames. It is false when the session has no handle to run by.
func NewGraph(calls Calls, session *ort.Session, inNames, outNames [][]byte, inputs []*ort.Value) (*Graph, bool) {
	if session == nil || len(inNames) != len(inputs) {
		return nil, false
	}
	// A Session and a Value each begin with their runtime handle.
	handle := *(*uintptr)(unsafe.Pointer(session))
	if handle == 0 {
		return nil, false
	}
	return newGraphOf(calls, handle, inNames, outNames, getValueHandles(inputs)), true
}

func newGraphOf(calls Calls, session uintptr, inNames, outNames [][]byte, inputs []uintptr) *Graph {
	r := &Graph{
		calls:    calls,
		session:  session,
		inNames:  make([]unsafe.Pointer, len(inNames)),
		outNames: make([]unsafe.Pointer, len(outNames)),
		inputs:   inputs,
		outputs:  make([]uintptr, len(outNames)),
	}
	for i, name := range inNames {
		r.inNames[i] = unsafe.Pointer(&name[0])
	}
	for i, name := range outNames {
		r.outNames[i] = unsafe.Pointer(&name[0])
	}
	return r
}

func getValueHandles(values []*ort.Value) []uintptr {
	out := make([]uintptr, len(values))
	for i, value := range values {
		// A Value begins with its runtime handle.
		out[i] = *(*uintptr)(unsafe.Pointer(value))
	}
	return out
}

// Run runs the graph once. The outputs it writes stay valid until the next run
// or close.
func (r *Graph) Run() error {
	// The name and handle arrays and the name bytes are kept alive by r for the call.
	status := r.calls.Run(
		r.session,
		0,
		uintptr(unsafe.Pointer(&r.inNames[0])),
		uintptr(unsafe.Pointer(&r.inputs[0])),
		uintptr(len(r.inputs)),
		uintptr(unsafe.Pointer(&r.outNames[0])),
		uintptr(len(r.outNames)),
		uintptr(unsafe.Pointer(&r.outputs[0])),
	)
	runtime.KeepAlive(r)
	return r.calls.Check(status)
}

// CopyOutput copies the first len(dst) values of output i into dst. It fails
// when the output holds fewer.
func (r *Graph) CopyOutput(i int, dst []float32) error {
	if err := r.calls.Check(r.calls.GetData(r.outputs[i], uintptr(unsafe.Pointer(&r.data)))); err != nil {
		return err
	}
	if r.data == nil {
		return fmt.Errorf("the output %d holds no data", i)
	}
	// The output was sized by the first position and is not released until close.
	copy(dst, unsafe.Slice((*float32)(r.data), len(dst)))
	return nil
}

// Close releases every output the runtime created.
func (r *Graph) Close() {
	for i, value := range r.outputs {
		if value != 0 {
			r.calls.Release(value)
			r.outputs[i] = 0
		}
	}
}
