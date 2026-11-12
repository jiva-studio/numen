// Package segmenter finds the speech segments in a recording.
package segmenter

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/ortraw"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/signal"
)

// The input and output tensor names of the segmenter graph.
const (
	speechInput = "input"
	speechState = "state"
	speechRate  = "sr"
	speechScore = "output"
	speechAfter = "stateN"
)

// The model takes the recording in windows of this many samples, and carries
// this much state from one to the next. At 16 kHz a window is 32 milliseconds,
// and that is how finely a span can be cut.
const (
	speechWindow = 512
	speechMemory = 2 * 128
)

// Segment is one stretch of a recording that carries speech: its samples and
// the milliseconds it spans.
type Segment struct {
	Samples []float32
	From    int
	To      int
}

// A Segmenter cuts recordings with one speech model. Cutting runs under lock,
// which the owner shares with whatever else uses the model's runtime.
type Segmenter struct {
	session *ort.Session
	model   settings.SegmenterModel
	lock    sync.Locker
	runner  *ortraw.Runner
}

// New returns a Segmenter that runs session with the parameters of model, calling the runtime
// directly through runner where it offers the entry points.
func New(session *ort.Session, model settings.SegmenterModel, lock sync.Locker, runner *ortraw.Runner) *Segmenter {
	return &Segmenter{session: session, model: model, lock: lock, runner: runner}
}

// Split cuts one recording into the segments that carry speech.
//
// The model scores one window at a time and carries its state across windows.
// A run of windows scoring above the threshold is a segment, closed by the
// silence after it and padded at each end.
func (s *Segmenter) Split(ctx context.Context, sound []float32) ([]Segment, error) {
	scores, err := s.scoreSpeech(ctx, sound)
	if err != nil {
		return nil, err
	}

	windows := func(ms int) int { return ms * signal.SampleRate / (1000 * speechWindow) }
	found := joinShortSpans(
		getSpans(scores, s.model.GetThreshold(),
			windows(s.model.GetSilenceLength()), windows(s.model.GetPadding()),
			windows(s.model.GetLongestSegment()), windows(s.model.GetShortestSegment())),
		windows(s.model.GetShortestPause()), windows(s.model.GetLongestSegment()),
	)

	var out []Segment
	for _, one := range found {
		from := min(one.From*speechWindow, len(sound))
		to := min(one.To*speechWindow, len(sound))
		out = append(out, Segment{
			Samples: sound[from:to],
			From:    signal.GetMillis(from, signal.SampleRate),
			To:      signal.GetMillis(to, signal.SampleRate),
		})
	}
	return out, nil
}

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

func (s *Segmenter) scoreSpeech(ctx context.Context, sound []float32) ([]float32, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	pass, err := newSpeechPass(sound, ort.NewTensor)
	if err != nil {
		return nil, err
	}
	defer pass.destroy()

	var out []float32
	if calls, ok := s.runner.GetCalls(); ok && ortraw.GetSessionHandle(s.session) != 0 {
		out, err = s.scoreNative(ctx, pass, calls)
	} else {
		out, err = s.scoreManaged(ctx, pass)
	}
	if err != nil {
		return nil, err
	}
	pass.keepAlive()
	return out, nil
}

// speechPass is the buffers and tensors the speech model reads while one
// recording is scored: the window being scored, the state carried from one
// window to the next, and the rate.
type speechPass struct {
	sound  []float32
	window []float32
	state  []float32
	rate   []int64
	count  int

	input, memory, sampling *ort.Value
}

// tensorMaker makes a tensor of the shape over the data.
type tensorMaker func(shape []int64, data any) (*ort.Value, error)

func newSpeechPass(sound []float32, newTensor tensorMaker) (*speechPass, error) {
	p := &speechPass{
		sound:  sound,
		window: make([]float32, speechWindow),
		state:  make([]float32, speechMemory),
		rate:   []int64{signal.SampleRate},
		count:  (len(sound) + speechWindow - 1) / speechWindow,

		input:    nil,
		memory:   nil,
		sampling: nil,
	}
	var err error
	if p.input, err = newTensor([]int64{1, speechWindow}, p.window); err != nil {
		return nil, err
	}
	if p.memory, err = newTensor([]int64{2, 1, speechMemory / 2}, p.state); err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Destroy on uninitialized test tensors has no side effect"
		p.destroy()
		return nil, err
	}
	if p.sampling, err = newTensor(nil, p.rate); err != nil {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="Destroy on uninitialized test tensors has no side effect"
		p.destroy()
		return nil, err
	}
	return p, nil
}

func (p *speechPass) destroy() {
	// gomutants:disable-next-line RANGE_BREAK,BRANCH_IF,STATEMENT_REMOVE reason="Destroy on uninitialized test tensors has no side effect"
	for _, tensor := range []*ort.Value{p.input, p.memory, p.sampling} {
		// gomutants:disable-next-line BRANCH_IF reason="Destroy on uninitialized test tensors has no side effect"
		if tensor != nil {
			// gomutants:disable-next-line STATEMENT_REMOVE reason="Destroy on uninitialized test tensors has no side effect"
			tensor.Destroy()
		}
	}
}

func (p *speechPass) load(i int) {
	clear(p.window)
	// gomutants:disable-next-line INTEGER_INCREMENT reason="copy fills at most the window length"
	copy(p.window, p.sound[i*speechWindow:min((i+1)*speechWindow, len(p.sound))])
}

func (p *speechPass) keepAlive() {
	runtime.KeepAlive(p.window)
	runtime.KeepAlive(p.state)
	runtime.KeepAlive(p.rate)
}

func (s *Segmenter) scoreNative(ctx context.Context, pass *speechPass, calls ortraw.Calls) ([]float32, error) {
	names := ortraw.MakeCNames(speechInput, speechState, speechRate, speechScore, speechAfter)
	run := &speechRun{
		calls:   calls,
		session: ortraw.GetSessionHandle(s.session),
		inNames: [3]unsafe.Pointer{
			unsafe.Pointer(&names[0][0]),
			unsafe.Pointer(&names[1][0]),
			unsafe.Pointer(&names[2][0]),
		},
		outNames: [2]unsafe.Pointer{
			unsafe.Pointer(&names[3][0]),
			unsafe.Pointer(&names[4][0]),
		},
		inputs: [3]uintptr{
			ortraw.GetValueHandle(pass.input),
			ortraw.GetValueHandle(pass.memory),
			ortraw.GetValueHandle(pass.sampling),
		},
		outputs: [2]uintptr{},
	}
	out := make([]float32, 0, pass.count)
	for i := range pass.count {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pass.load(i)
		score, err := run.score(pass.state)
		if err != nil {
			return nil, err
		}
		out = append(out, score)
	}
	return out, nil
}

func (s *Segmenter) scoreManaged(ctx context.Context, pass *speechPass) ([]float32, error) {
	inputs := map[string]*ort.Value{
		speechInput: pass.input,
		speechState: pass.memory,
		speechRate:  pass.sampling,
	}
	out := make([]float32, 0, pass.count)
	for i := range pass.count {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		pass.load(i)
		score, err := s.scoreWindow(inputs, pass.state)
		if err != nil {
			return nil, err
		}
		out = append(out, score)
	}
	return out, nil
}

// speechRun is the speech model run window after window by the runtime's own
// entry points, over the same inputs.
type speechRun struct {
	calls    ortraw.Calls
	session  uintptr
	inNames  [3]unsafe.Pointer
	outNames [2]unsafe.Pointer
	inputs   [3]uintptr
	outputs  [2]uintptr
}

func (r *speechRun) score(state []float32) (float32, error) {
	r.outputs = [2]uintptr{}
	status := r.calls.Run(
		r.session,
		0,
		uintptr(unsafe.Pointer(&r.inNames[0])),
		uintptr(unsafe.Pointer(&r.inputs[0])),
		3,
		uintptr(unsafe.Pointer(&r.outNames[0])),
		2,
		uintptr(unsafe.Pointer(&r.outputs[0])),
	)
	if err := r.calls.Check(status); err != nil {
		return 0, err
	}
	defer r.calls.Release(r.outputs[0])
	defer r.calls.Release(r.outputs[1])

	var scorePtr, statePtr unsafe.Pointer
	if err := r.calls.Check(r.calls.GetData(r.outputs[0], uintptr(unsafe.Pointer(&scorePtr)))); err != nil {
		return 0, err
	}
	if err := r.calls.Check(r.calls.GetData(r.outputs[1], uintptr(unsafe.Pointer(&statePtr)))); err != nil {
		return 0, err
	}
	copy(state, unsafe.Slice((*float32)(statePtr), speechMemory))
	return *(*float32)(scorePtr), nil
}

func (s *Segmenter) scoreWindow(inputs map[string]*ort.Value, dst []float32) (float32, error) {
	out, err := s.session.Run(inputs)
	if err != nil {
		return 0, err
	}
	defer func() {
		for _, v := range out {
			v.Destroy()
		}
	}()

	response, ok := out[speechScore]
	if !ok {
		return 0, fmt.Errorf("the speech model answered with %v and not %s", ortraw.GetNames(out), speechScore)
	}
	score, err := ort.GetTensorData[float32](response)
	if err != nil {
		return 0, err
	}
	if len(score) == 0 {
		return 0, fmt.Errorf("the speech model answered with no score")
	}
	after, ok := out[speechAfter]
	if !ok {
		return 0, fmt.Errorf("the speech model answered with %v and not %s", ortraw.GetNames(out), speechAfter)
	}
	raw, err := ort.GetTensorData[float32](after)
	if err != nil {
		return 0, err
	}
	copy(dst, raw)
	return score[0], nil
}

// A span is a run of the recording, counted in windows: from the first, up
// to but not including the last.
type span struct {
	From, To int
}

func getSpans(scores []float32, threshold float32, silenceWindows, pad, longest, shortest int) []span {
	out := findSpeechSpans(scores, threshold, silenceWindows)
	wider := widenSpans(out, pad, len(scores))

	var trimmed []span
	for _, one := range wider {
		trimmed = append(trimmed, splitLongSpan(one, scores, longest)...)
	}

	out = out[:0]
	for _, one := range trimmed {
		if one.To-one.From >= shortest {
			out = append(out, one)
		}
	}
	return out
}

func findSpeechSpans(scores []float32, threshold float32, silenceWindows int) []span {
	var out []span
	// gomutants:disable-next-line INTEGER_INCREMENT,INTEGER_DECREMENT,INVERT_NEGATIVES reason="any negative number marks the span as closed, and last is always set on the first speech window before it is read"
	open, last := -1, -1
	for i, score := range scores {
		if score >= threshold {
			if open < 0 {
				open = i
			}
			last = i
			continue
		}
		if open >= 0 && i-last > silenceWindows {
			out = append(out, span{open, last + 1})
			// gomutants:disable-next-line INTEGER_INCREMENT reason="any negative number marks the span as closed"
			open = -1
		}
	}
	if open >= 0 {
		out = append(out, span{open, last + 1})
	}
	return out
}

func widenSpans(found []span, pad, count int) []span {
	var wider []span
	for _, one := range found {
		one.From = max(one.From-pad, 0)
		one.To = min(one.To+pad, count)
		if n := len(wider); n > 0 && one.From <= wider[n-1].To {
			wider[n-1].To = one.To
			continue
		}
		wider = append(wider, one)
	}
	return wider
}

func splitLongSpan(one span, scores []float32, longest int) []span {
	if longest <= 1 {
		return []span{one}
	}
	var out []span
	for one.To-one.From > longest {
		pos := findQuietestIndex(scores, one.From+longest/2, one.From+longest)
		out = append(out, span{one.From, pos})
		one.From = pos
	}
	return append(out, one)
}

func findQuietestIndex(scores []float32, from, to int) int {
	from, to = max(from, 0), min(to, len(scores))
	pos := from
	for i := from; i < to; i++ {
		if scores[i] < scores[pos] {
			pos = i
		}
	}
	return pos
}
