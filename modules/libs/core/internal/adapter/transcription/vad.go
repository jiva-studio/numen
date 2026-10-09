package transcription

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The names the segmenter's graph gives what it is asked and what it answers.
const (
	speechInput = "input"
	speechState = "state"
	speechRate  = "sr"
	speechScore = "output"
	speechAfter = "stateN"
)

var (
	speechInputCStr = []byte("input\x00")
	speechStateCStr = []byte("state\x00")
	speechRateCStr  = []byte("sr\x00")
	speechScoreCStr = []byte("output\x00")
	speechAfterCStr = []byte("stateN\x00")
)

var (
	runnerMu sync.Mutex
	ortRun   func(
		session, runOptions, inputNames, inputValues, inputCount, outputNames, outputCount, outputValues uintptr,
	) uintptr
	ortGetTensorMutableData func(value, out uintptr) uintptr
	ortReleaseValue         func(value uintptr)
	ortReleaseStatus        func(status uintptr)
	ortGetErrorCode         func(status uintptr) int32
	ortGetErrorMessage      func(status uintptr) unsafe.Pointer
)

type ortAPIBase struct {
	GetAPI           uintptr
	GetVersionString uintptr
}

type ortAPITable struct {
	_                    uintptr
	GetErrorCode         uintptr
	GetErrorMessage      uintptr
	_                    [6]uintptr
	Run                  uintptr
	_                    [41]uintptr
	GetTensorMutableData uintptr
	_                    [41]uintptr
	ReleaseStatus        uintptr
	_                    [2]uintptr
	ReleaseValue         uintptr
}

func init() {
	onnxruntime.Register(func(at string) {
		_ = initRunner(at)
	})
}

func initRunner(at string) error {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	if ortRun != nil {
		return nil
	}
	if at == "" {
		return fmt.Errorf("onnx runtime not loaded")
	}

	handle, err := onnxruntime.OpenLibrary(at)
	if err != nil {
		return err
	}

	var ortGetAPIBase func() *ortAPIBase
	purego.RegisterLibFunc(&ortGetAPIBase, handle, "OrtGetApiBase")
	if ortGetAPIBase == nil {
		return fmt.Errorf("symbol OrtGetApiBase not found")
	}

	apiBase := ortGetAPIBase()
	if apiBase == nil {
		return fmt.Errorf("OrtGetApiBase returned nil")
	}

	var getAPI func(uint32) *ortAPITable
	purego.RegisterFunc(&getAPI, apiBase.GetAPI)
	api := getAPI(23)
	if api == nil || api.Run == 0 {
		return fmt.Errorf("failed to get OrtApi")
	}

	purego.RegisterFunc(&ortRun, api.Run)
	purego.RegisterFunc(&ortGetTensorMutableData, api.GetTensorMutableData)
	purego.RegisterFunc(&ortReleaseValue, api.ReleaseValue)
	purego.RegisterFunc(&ortReleaseStatus, api.ReleaseStatus)
	purego.RegisterFunc(&ortGetErrorCode, api.GetErrorCode)
	purego.RegisterFunc(&ortGetErrorMessage, api.GetErrorMessage)
	return nil
}

func checkStatus(status uintptr) error {
	if status == 0 {
		return nil
	}
	defer ortReleaseStatus(status)
	code := ortGetErrorCode(status)
	msgPtr := ortGetErrorMessage(status)
	var msg string
	if msgPtr != nil {
		var b []byte
		for p := (*byte)(msgPtr); *p != 0; p = (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + 1)) {
			b = append(b, *p)
		}
		msg = string(b)
	}
	return fmt.Errorf("onnxruntime error [code %d]: %s", code, msg)
}

// The model takes the recording in windows of this many samples, and carries
// this much state from one to the next. At 16 kHz a window is 32 milliseconds,
// and that is how finely a span can be cut.
const (
	speechWindow = 512
	speechMemory = 2 * 128
)

// segments cuts one recording into the segments that carry speech.
//
// The model answers for one window at a time, carrying what it made of the
// windows before. A run of windows it is sure enough about is a segment, closed
// by the quiet after it and widened a little at each end.
func (t *Transcriber) segments(ctx context.Context, sound []float32) ([]port.Audio, error) {
	scores, err := t.scoreSpeech(ctx, sound)
	if err != nil {
		return nil, err
	}

	windows := func(ms int) int { return ms * sampleRate / (1000 * speechWindow) }
	found := joinShortSpans(
		getSpans(scores, t.cutting.threshold(),
			windows(t.cutting.silence()), windows(t.cutting.pad()),
			windows(t.cutting.getLongestSegment()), windows(t.cutting.getShortestSegment())),
		windows(t.cutting.getShortestPause()), windows(t.cutting.getLongestSegment()),
	)

	var out []port.Audio
	for _, one := range found {
		from := min(one.From*speechWindow, len(sound))
		to := min(one.To*speechWindow, len(sound))
		out = append(out, port.Audio{
			Samples: sound[from:to],
			From:    millis(from, sampleRate),
			To:      millis(to, sampleRate),
		})
	}
	return out, nil
}

// joinShortSpans puts a span too short to stand on its own together with the one
// after it, up to the longest a span may run to.
//
// A person pausing in the middle of a sentence closes a span, and what comes
// back is a line holding one word. A line is a thing somebody reads, and the
// model hears a sentence better than it hears a word out of one.
func joinShortSpans(found []span, least, longest int) []span {
	out := make([]span, 0, len(found))
	for _, one := range found {
		if len(out) == 0 {
			out = append(out, one)
			continue
		}
		last := &out[len(out)-1]
		short := last.To-last.From < least
		if short && one.To-last.From <= longest {
			last.To = one.To
			continue
		}
		out = append(out, one)
	}
	return out
}

// scoreSpeech is how sure the model is that each window of the recording carries
// speech. A last window short of the model's own is filled out with silence.
func (t *Transcriber) scoreSpeech(ctx context.Context, sound []float32) ([]float32, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := make([]float32, speechMemory)
	rate := []int64{sampleRate}
	window := make([]float32, speechWindow)

	inputTensor, err := ort.NewTensor([]int64{1, speechWindow}, window)
	if err != nil {
		return nil, err
	}
	defer inputTensor.Destroy()

	stateTensor, err := ort.NewTensor([]int64{2, 1, speechMemory / 2}, state)
	if err != nil {
		return nil, err
	}
	defer stateTensor.Destroy()

	rateTensor, err := ort.NewTensor(nil, rate)
	if err != nil {
		return nil, err
	}
	defer rateTensor.Destroy()

	count := (len(sound) + speechWindow - 1) / speechWindow
	out := make([]float32, 0, count)

	var sessionHandle uintptr
	if t.segmenter != nil {
		sessionHandle = *(*uintptr)(unsafe.Pointer(t.segmenter))
	}

	runnerMu.Lock()
	runFn := ortRun
	getMutableDataFn := ortGetTensorMutableData
	releaseValFn := ortReleaseValue
	runnerMu.Unlock()

	if runFn != nil && sessionHandle != 0 {
		inputNames := [3]unsafe.Pointer{
			unsafe.Pointer(&speechInputCStr[0]),
			unsafe.Pointer(&speechStateCStr[0]),
			unsafe.Pointer(&speechRateCStr[0]),
		}
		outputNames := [2]unsafe.Pointer{
			unsafe.Pointer(&speechScoreCStr[0]),
			unsafe.Pointer(&speechAfterCStr[0]),
		}
		inputHandles := [3]uintptr{
			*(*uintptr)(unsafe.Pointer(inputTensor)),
			*(*uintptr)(unsafe.Pointer(stateTensor)),
			*(*uintptr)(unsafe.Pointer(rateTensor)),
		}
		var outputHandles [2]uintptr

		for i := 0; i < count; i++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			clear(window)
			copy(window, sound[i*speechWindow:min((i+1)*speechWindow, len(sound))])

			outputHandles[0] = 0
			outputHandles[1] = 0

			status := runFn(
				sessionHandle,
				0,
				uintptr(unsafe.Pointer(&inputNames[0])),
				uintptr(unsafe.Pointer(&inputHandles[0])),
				3,
				uintptr(unsafe.Pointer(&outputNames[0])),
				2,
				uintptr(unsafe.Pointer(&outputHandles[0])),
			)
			if err := checkStatus(status); err != nil {
				return nil, err
			}

			var scorePtr, statePtr unsafe.Pointer
			status = getMutableDataFn(outputHandles[0], uintptr(unsafe.Pointer(&scorePtr)))
			if err := checkStatus(status); err != nil {
				releaseValFn(outputHandles[0])
				releaseValFn(outputHandles[1])
				return nil, err
			}
			status = getMutableDataFn(outputHandles[1], uintptr(unsafe.Pointer(&statePtr)))
			if err := checkStatus(status); err != nil {
				releaseValFn(outputHandles[0])
				releaseValFn(outputHandles[1])
				return nil, err
			}

			score := *(*float32)(scorePtr)
			copy(state, unsafe.Slice((*float32)(statePtr), speechMemory))

			releaseValFn(outputHandles[0])
			releaseValFn(outputHandles[1])

			out = append(out, score)
		}
	} else {
		inputs := map[string]*ort.Value{
			speechInput: inputTensor,
			speechState: stateTensor,
			speechRate:  rateTensor,
		}
		for i := 0; i < count; i++ {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			clear(window)
			copy(window, sound[i*speechWindow:min((i+1)*speechWindow, len(sound))])

			score, err := t.listen(inputs, state)
			if err != nil {
				return nil, err
			}
			out = append(out, score)
		}
	}

	runtime.KeepAlive(window)
	runtime.KeepAlive(state)
	runtime.KeepAlive(rate)
	return out, nil
}

// listen is one window: how sure the model is, and the state it leaves behind.
func (t *Transcriber) listen(inputs map[string]*ort.Value, dst []float32) (float32, error) {
	out, err := t.segmenter.Run(inputs)
	if err != nil {
		return 0, err
	}

	said, ok := out[speechScore]
	if !ok {
		for _, v := range out {
			v.Destroy()
		}
		return 0, fmt.Errorf("the speech model answered with %v and not %s", names(out), speechScore)
	}
	score, err := ort.GetTensorData[float32](said)
	if err != nil {
		for _, v := range out {
			v.Destroy()
		}
		return 0, err
	}
	if len(score) == 0 {
		for _, v := range out {
			v.Destroy()
		}
		return 0, fmt.Errorf("the speech model answered with no score")
	}
	after, ok := out[speechAfter]
	if !ok {
		for _, v := range out {
			v.Destroy()
		}
		return 0, fmt.Errorf("the speech model answered with %v and not %s", names(out), speechAfter)
	}
	raw, err := ort.GetTensorData[float32](after)
	if err != nil {
		for _, v := range out {
			v.Destroy()
		}
		return 0, err
	}
	copy(dst, raw)
	for _, v := range out {
		v.Destroy()
	}
	return score[0], nil
}

// A span is a run of the recording, counted in windows: from the first, up
// to but not including the last.
type span struct {
	From, To int
}

// getSpans are the spans of speech a run of scores holds.
//
// A span is opened by a window the model is sure enough about and closed by
// quiet windows enough after it, so that the pause between two words does not
// cut a sentence in half. Each is then widened by pad at both ends, and two
// that now meet are one.
func getSpans(scores []float32, threshold float32, silence, pad, longest, shortest int) []span {
	var out []span
	open, last := -1, -1
	for i, score := range scores {
		if score >= threshold {
			if open < 0 {
				open = i
			}
			last = i
			continue
		}
		if open >= 0 && i-last > silence {
			out = append(out, span{open, last + 1})
			open = -1
		}
	}
	if open >= 0 {
		out = append(out, span{open, last + 1})
	}

	var wider []span
	for _, one := range out {
		one.From = max(one.From-pad, 0)
		one.To = min(one.To+pad, len(scores))
		if n := len(wider); n > 0 && one.From <= wider[n-1].To {
			wider[n-1].To = one.To
			continue
		}
		wider = append(wider, one)
	}

	var cut []span
	for _, one := range wider {
		cut = append(cut, splitLongSpan(one, scores, longest)...)
	}

	out = out[:0]
	for _, one := range cut {
		if one.To-one.From >= shortest {
			out = append(out, one)
		}
	}
	return out
}

// splitLongSpan cuts a span that runs on too long into pieces the model is given
// one at a time. Each cut falls on the quietest window of the second half of
// what is left, so that a sentence is broken where the speaker paused.
func splitLongSpan(one span, scores []float32, longest int) []span {
	if longest <= 1 {
		return []span{one}
	}
	var out []span
	for one.To-one.From > longest {
		at := findQuietest(scores, one.From+longest/2, one.From+longest)
		out = append(out, span{one.From, at})
		one.From = at
	}
	return append(out, one)
}

// findQuietest is where the lowest score between two windows stands.
func findQuietest(scores []float32, from, to int) int {
	from, to = max(from, 0), min(to, len(scores))
	at := from
	for i := from; i < to; i++ {
		if scores[i] < scores[at] {
			at = i
		}
	}
	return at
}
