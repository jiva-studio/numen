// Package engine converts a recording into text with models running on
// this machine.
//
// A recording passes through two models: one detects the segments that contain
// speech, and one transcribes each segment. They run through ONNX Runtime, which
// is loaded by name at run time, so the package builds with CGO_ENABLED=0 and
// cross-compiles from any host to any target.
//
// The transducer is exported as three graphs. The encoder converts a spectrogram
// into frames, the predictor carries the state of the tokens emitted so far, and
// the joiner combines both into one token and the number of frames to advance.
package engine

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/features"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/ortraw"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/segmenter"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/tokens"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/transducer"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/settings"
)

// The input and output names of the graphs.
const (
	inputSignal  = "audio_signal"
	inputLength  = "length"
	outputFrames = "outputs"
	outputLength = "encoded_lengths"

	inputTargets = "targets"
	inputTargetN = "target_length"
	inputState   = "states.1"
	inputCell    = "onnx::Slice_3"
	outputState  = "states"
	outputCell   = "162"

	inputEncoded = "encoder_outputs"
	inputDecoded = "decoder_outputs"
)

// What one frame of each graph is: the encoder's channels, the predictor's
// hidden width, and the two layers of its state.
const (
	encoded = 1024
	hidden  = 640
	layers  = 2
)

func getDecoderInputNames() [][]byte {
	return ortraw.MakeCNames(inputTargets, inputTargetN, inputState, inputCell)
}

func getDecoderOutputs() [][]byte {
	return ortraw.MakeCNames(outputFrames, outputState, outputCell)
}

func getJoinerInputNames() [][]byte { return ortraw.MakeCNames(inputEncoded, inputDecoded) }

func getJoinerOutputs() [][]byte { return ortraw.MakeCNames(outputFrames) }

// blankPiece is the token the model emits when a frame carries no token. It is
// the last entry in the pieces file, after every token the model can emit.
const blankPiece = "<blk>"

// A Transcriber is the set of models this machine transcribes a recording
// with.
type Transcriber struct {
	encoder *ort.Session
	decoder *ort.Session
	joiner  *ort.Session

	pieces tokens.Pieces
	blank  int

	segmenter *ort.Session
	cutter    *segmenter.Segmenter

	runner *ortraw.Runner

	workers int

	mu sync.Mutex
}

// Open loads the models through the runtime and compiles them, reading the weights. The result is
// used for any number of recordings until it is closed.
func Open(ctx context.Context, runtime Runtime, cfg settings.Config) (*Transcriber, error) {
	engine, found, err := locateModels(ctx, runtime.library, cfg)
	if err != nil {
		return nil, err
	}

	pieces, err := tokens.Read(found.tokens)
	if err != nil {
		return nil, err
	}
	blank := pieces.GetIndex(blankPiece)
	if blank < 0 {
		return nil, fmt.Errorf("%s names no %s", found.tokens, blankPiece)
	}

	out := &Transcriber{
		encoder:   nil,
		decoder:   nil,
		joiner:    nil,
		pieces:    pieces,
		blank:     blank,
		segmenter: nil,
		cutter:    nil,
		runner:    runtime.runner,
		workers:   cfg.Workers,
		mu:        sync.Mutex{},
	}
	threads := getSessionThreads(cfg)
	for _, one := range []struct {
		dst     **ort.Session
		path    string
		kind    string
		threads int
	}{
		{&out.encoder, found.encoder, "encoder", threads.encoder},
		{&out.decoder, found.decoder, "decoder", threads.decoder},
		{&out.joiner, found.joiner, "joiner", threads.joiner},
		{&out.segmenter, found.segmenter, "segmenter model", threads.segmenter},
	} {
		session, err := openSession(runtime.library, engine, one.path, one.threads)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("the %s %s: %w", one.kind, one.path, err), out.Close())
		}
		*one.dst = session
	}
	out.cutter = segmenter.New(out.segmenter, cfg.Segmenter, &out.mu, runtime.runner)
	return out, nil
}

func openSession(library *onnx.Library, engine *ort.Engine, path string, threads int) (*ort.Session, error) {
	options, _, err := library.NewSessionOptions(engine, onnx.SessionSettings{
		Provider:     onnx.GetSessionProvider(),
		Threads:      threads,
		ComputeUnits: onnx.ComputeUnitsDefault,
		ModelPath:    path,
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()
	return onnx.NewSession(engine, path, options)
}

// Close releases the models this transcriber loaded. The runtime they ran on
// is the process's and stays.
func (t *Transcriber) Close() error {
	for _, session := range []*ort.Session{t.encoder, t.decoder, t.joiner, t.segmenter} {
		if session != nil {
			session.Destroy()
		}
	}
	return nil
}

// Transcribe is one stretch of speech, as the words it carries.
func (t *Transcriber) Transcribe(ctx context.Context, audio Audio) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	feature, frames := features.GetLogMel(audio.Samples, t.workers)
	if frames < 2 {
		return "", nil
	}

	signal, err := ort.NewTensor([]int64{1, features.MelBands, int64(frames)}, feature)
	if err != nil {
		return "", err
	}
	defer signal.Destroy()
	span := []int64{int64(frames)}
	length, err := ort.NewTensor([]int64{1}, span)
	if err != nil {
		return "", err
	}
	defer length.Destroy()

	out, err := t.encoder.Run(map[string]*ort.Value{inputSignal: signal, inputLength: length})
	// The library keeps a pointer into these and nothing else does, so they are
	// held until the run is over.
	runtime.KeepAlive(feature)
	runtime.KeepAlive(span)
	if err != nil {
		return "", err
	}
	for _, v := range out {
		defer v.Destroy()
	}

	response, err := t.decode(ctx, out)
	if err != nil {
		return "", err
	}
	return t.pieces.GetText(response), nil
}

func (t *Transcriber) decode(ctx context.Context, out map[string]*ort.Value) ([]int, error) {
	return t.decodeWith(ctx, out, true)
}

func (t *Transcriber) decodeWith(ctx context.Context, out map[string]*ort.Value, useRaw bool) ([]int, error) {
	frames, err := readFrames(out)
	if err != nil {
		return nil, err
	}
	buf, err := newDecodeBuffers()
	if err != nil {
		return nil, err
	}
	defer buf.destroy()

	decoderInputs, joinerInputs := buf.getDecoderInputs(), buf.getJoinerInputs()
	decoderRun, joinerRun := t.makeRawRuns(buf, useRaw)
	if decoderRun != nil {
		defer decoderRun.Close()
	}
	if joinerRun != nil {
		defer joinerRun.Close()
	}
	predicted, joined := false, false
	var scores []float32

	result, err := transducer.Decoder{
		Frames:  frames.width,
		Blank:   t.blank,
		Encoded: encoded,
		Encoder: func(pos int, dst []float32) []float32 {
			frames.copyFrame(pos, dst)
			return dst
		},
		Predictor: func(token int) ([]float32, error) {
			buf.target[0] = int32(token)
			if decoderRun == nil || !predicted {
				predicted = true
				return t.predict(decoderInputs, buf.prediction, buf.state, buf.cell)
			}
			return t.predictRaw(decoderRun, buf.prediction, buf.state, buf.cell)
		},
		Joint: func(encodedFrame, _ []float32) ([]float32, error) {
			var err error
			if joinerRun == nil || !joined {
				joined = true
				scores, err = t.computeJoint(joinerInputs, buf.frame, encodedFrame, scores)
			} else {
				scores, err = t.computeJointRaw(joinerRun, buf.frame, encodedFrame, scores)
			}
			return scores, err
		},
	}.Decode(ctx)

	buf.keepAlive()
	return result, err
}

// encodedFrames are the frames the encoder wrote, channel after channel, each
// holding every frame, and how many of them carry the stretch.
type encodedFrames struct {
	data   []float32
	stride int
	width  int
}

func readFrames(out map[string]*ort.Value) (encodedFrames, error) {
	frames, ok := out[outputFrames]
	if !ok {
		return encodedFrames{}, fmt.Errorf("the encoder answered with %v and not %s", ortraw.GetNames(out), outputFrames)
	}
	shape, err := frames.GetShape()
	if err != nil {
		return encodedFrames{}, err
	}
	if len(shape) != 3 || shape[1] != encoded {
		return encodedFrames{}, fmt.Errorf("the encoder answered with frames of %v", shape)
	}
	data, err := ort.GetTensorData[float32](frames)
	if err != nil {
		return encodedFrames{}, err
	}
	width := int(shape[2])
	if counted, ok := out[outputLength]; ok {
		if n, ok := readFrameCount(counted); ok && n < width {
			width = n
		}
	}
	return encodedFrames{data: data, stride: int(shape[2]), width: width}, nil
}

func readFrameCount(counted *ort.Value) (int, bool) {
	lengths, err := ort.GetTensorData[int64](counted)
	if err != nil || len(lengths) == 0 || lengths[0] < 0 {
		return 0, false
	}
	return int(lengths[0]), true
}

func (f encodedFrames) copyFrame(pos int, dst []float32) {
	for c := range dst {
		dst[c] = f.data[c*f.stride+pos]
	}
}

// decodeBuffers are the buffers and tensors the predictor and the joiner read
// and write while one stretch is decoded.
type decodeBuffers struct {
	target     []int32
	count      []int32
	state      []float32
	cell       []float32
	prediction []float32
	frame      []float32

	targetTensor, countTensor, stateTensor, cellTensor, frameTensor, predictionTensor *ort.Value
}

func newDecodeBuffers() (*decodeBuffers, error) {
	b := &decodeBuffers{
		target:     []int32{0},
		count:      []int32{1},
		state:      make([]float32, layers*hidden),
		cell:       make([]float32, layers*hidden),
		prediction: make([]float32, hidden),
		frame:      make([]float32, encoded),

		targetTensor:     nil,
		countTensor:      nil,
		stateTensor:      nil,
		cellTensor:       nil,
		frameTensor:      nil,
		predictionTensor: nil,
	}
	for _, one := range []struct {
		into   **ort.Value
		create func() (*ort.Value, error)
	}{
		{&b.targetTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{1, 1}, b.target) }},
		{&b.countTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{1}, b.count) }},
		{&b.stateTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{layers, 1, hidden}, b.state) }},
		{&b.cellTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{layers, 1, hidden}, b.cell) }},
		{&b.frameTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{1, encoded, 1}, b.frame) }},
		{&b.predictionTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{1, hidden, 1}, b.prediction) }},
	} {
		tensor, err := one.create()
		if err != nil {
			b.destroy()
			return nil, err
		}
		*one.into = tensor
	}
	return b, nil
}

func (b *decodeBuffers) destroy() {
	for _, tensor := range []*ort.Value{
		b.targetTensor, b.countTensor, b.stateTensor, b.cellTensor, b.frameTensor, b.predictionTensor,
	} {
		if tensor != nil {
			tensor.Destroy()
		}
	}
}

func (b *decodeBuffers) keepAlive() {
	runtime.KeepAlive(b.target)
	runtime.KeepAlive(b.count)
	runtime.KeepAlive(b.state)
	runtime.KeepAlive(b.cell)
	runtime.KeepAlive(b.prediction)
	runtime.KeepAlive(b.frame)
}

func (b *decodeBuffers) getDecoderInputs() map[string]*ort.Value {
	return map[string]*ort.Value{
		inputTargets: b.targetTensor,
		inputTargetN: b.countTensor,
		inputState:   b.stateTensor,
		inputCell:    b.cellTensor,
	}
}

func (b *decodeBuffers) getJoinerInputs() map[string]*ort.Value {
	return map[string]*ort.Value{
		inputEncoded: b.frameTensor,
		inputDecoded: b.predictionTensor,
	}
}

func (t *Transcriber) makeRawRuns(b *decodeBuffers, useRaw bool) (decoderRun, joinerRun *ortraw.Graph) {
	calls, ok := t.runner.GetCalls()
	if !ok || !useRaw {
		return nil, nil
	}
	decoderRun, _ = ortraw.NewGraph(calls, t.decoder, getDecoderInputNames(), getDecoderOutputs(),
		[]*ort.Value{b.targetTensor, b.countTensor, b.stateTensor, b.cellTensor})
	joinerRun, _ = ortraw.NewGraph(calls, t.joiner, getJoinerInputNames(), getJoinerOutputs(),
		[]*ort.Value{b.frameTensor, b.predictionTensor})
	return decoderRun, joinerRun
}

func (t *Transcriber) predict(inputs map[string]*ort.Value, response, state, cell []float32) ([]float32, error) {
	out, err := t.decoder.Run(inputs)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, v := range out {
			v.Destroy()
		}
	}()

	for _, one := range [...]struct {
		into []float32
		name string
	}{
		{response, outputFrames},
		{state, outputState},
		{cell, outputCell},
	} {
		if err := copyPrediction(out, one.name, one.into); err != nil {
			return nil, err
		}
	}
	return response, nil
}

func copyPrediction(out map[string]*ort.Value, name string, dst []float32) error {
	value, ok := out[name]
	if !ok {
		return fmt.Errorf("the predictor answered with %v and not %s", ortraw.GetNames(out), name)
	}
	raw, err := ort.GetTensorData[float32](value)
	if err != nil {
		return err
	}
	if len(raw) < len(dst) {
		return fmt.Errorf("the predictor answered %s with %d values and not %d", name, len(raw), len(dst))
	}
	copy(dst, raw)
	return nil
}

func (t *Transcriber) predictRaw(run *ortraw.Graph, prediction, state, cell []float32) ([]float32, error) {
	if err := run.Run(); err != nil {
		return nil, err
	}
	for i, into := range [...][]float32{prediction, state, cell} {
		if err := run.CopyOutput(i, into); err != nil {
			return nil, err
		}
	}
	return prediction, nil
}

func (t *Transcriber) computeJointRaw(run *ortraw.Graph, frameDst, frame, scoresDst []float32) ([]float32, error) {
	copy(frameDst, frame)
	if err := run.Run(); err != nil {
		return nil, err
	}
	if err := run.CopyOutput(0, scoresDst); err != nil {
		return nil, err
	}
	return scoresDst, nil
}

func (t *Transcriber) computeJoint(inputs map[string]*ort.Value, frameDst, frame, scoresDst []float32) ([]float32, error) {
	copy(frameDst, frame)
	out, err := t.joiner.Run(inputs)
	if err != nil {
		return nil, err
	}

	value, ok := out[outputFrames]
	if !ok {
		for _, v := range out {
			v.Destroy()
		}
		return nil, fmt.Errorf("the joiner answered with %v and not %s", ortraw.GetNames(out), outputFrames)
	}
	raw, err := ort.GetTensorData[float32](value)
	if err != nil {
		for _, v := range out {
			v.Destroy()
		}
		return nil, err
	}
	if len(scoresDst) != len(raw) {
		scoresDst = make([]float32, len(raw))
	}
	copy(scoresDst, raw)
	for _, v := range out {
		v.Destroy()
	}
	return scoresDst, nil
}
