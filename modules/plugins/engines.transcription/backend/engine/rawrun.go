package engine

import (
	"fmt"
	"runtime"
	"unsafe"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// The names the predictor and the joiner are run by, as the C strings the
// runtime reads them from.
var (
	decoderInputNames = makeCNames(inputTargets, inputTargetN, inputState, inputCell)
	decoderOutputs    = makeCNames(outputFrames, outputState, outputCell)
	joinerInputNames  = makeCNames(inputEncoded, inputDecoded)
	joinerOutputs     = makeCNames(outputFrames)
)

func makeCNames(names ...string) [][]byte {
	out := make([][]byte, len(names))
	for i, name := range names {
		out[i] = append([]byte(name), 0)
	}
	return out
}

// rawCalls are the runtime entry points a rawRun uses.
type rawCalls struct {
	run     func(session, options, inNames, inValues, inCount, outNames, outCount, outValues uintptr) uintptr
	getData func(value, out uintptr) uintptr
	release func(value uintptr)
	check   func(status uintptr) error
}

// getRawCalls are the runtime entry points, or false when the runtime has not
// registered them.
func getRawCalls() (rawCalls, bool) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	calls := rawCalls{run: ortRun, getData: ortGetTensorMutableData, release: ortReleaseValue, check: checkStatus}
	return calls, calls.run != nil && calls.getData != nil && calls.release != nil
}

// A rawRun is one graph run over and over on the same input tensors. It owns
// its output values: the first run lets the runtime create them, every later
// run writes into them, and close releases them.
type rawRun struct {
	calls   rawCalls
	session uintptr

	inNames  []unsafe.Pointer
	outNames []unsafe.Pointer
	inputs   []uintptr
	outputs  []uintptr
	data     unsafe.Pointer
}

// newRawRun is a rawRun of session over inputs, which are given in the order of
// inNames. It is false when the session has no handle to run by.
func newRawRun(calls rawCalls, session *ort.Session, inNames, outNames [][]byte, inputs []*ort.Value) (*rawRun, bool) {
	if session == nil || len(inNames) != len(inputs) {
		return nil, false
	}
	// A Session and a Value each begin with their runtime handle.
	handle := *(*uintptr)(unsafe.Pointer(session))
	if handle == 0 {
		return nil, false
	}
	return newRawRunOf(calls, handle, inNames, outNames, getValueHandles(inputs)), true
}

func newRawRunOf(calls rawCalls, session uintptr, inNames, outNames [][]byte, inputs []uintptr) *rawRun {
	r := &rawRun{
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

// run runs the graph once. The outputs it writes stay valid until the next run
// or close.
func (r *rawRun) run() error {
	// The name and handle arrays and the name bytes are kept alive by r for the call.
	status := r.calls.run(
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
	return r.calls.check(status)
}

// copyOutput copies the first len(dst) values of output i into dst. It fails
// when the output holds fewer.
func (r *rawRun) copyOutput(i int, dst []float32) error {
	if err := r.calls.check(r.calls.getData(r.outputs[i], uintptr(unsafe.Pointer(&r.data)))); err != nil {
		return err
	}
	if r.data == nil {
		return fmt.Errorf("the output %d holds no data", i)
	}
	// The output was sized by the first position and is not released until close.
	copy(dst, unsafe.Slice((*float32)(r.data), len(dst)))
	return nil
}

// close releases every output the runtime created.
func (r *rawRun) close() {
	for i, value := range r.outputs {
		if value != 0 {
			r.calls.release(value)
			r.outputs[i] = 0
		}
	}
}
