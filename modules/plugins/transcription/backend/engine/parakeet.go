// Package transcription turns a recording into words with models running on
// this machine.
//
// A recording goes through two of them: one finds the stretches that carry
// speech, and one hears what each stretch says. They are run through ONNX
// Runtime, reached by name at run time, so this builds with CGO_ENABLED=0 and
// cross-compiles from any machine to any other.
//
// The transducer is exported as three graphs. An encoder turns a spectrogram
// into frames, a predictor carries what has been said so far, and a joiner puts
// the two together into one token and how many frames to step on by.
package engine

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
	"github.com/jiva-studio/numen/modules/plugins/transcription/backend/onnxruntime"
)

// The names the graphs give what they are asked and what they answer.
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

// blankPiece is what the model says when a frame carries no token. It stands
// last in the file of pieces, after every token the model can write.
const blankPiece = "<blk>"

// A Transcriber is the models this machine hears a recording with.
type Transcriber struct {
	encoder *ort.Session
	decoder *ort.Session
	joiner  *ort.Session

	pieces pieces
	blank  int

	// segmenter is the model that finds the stretches, and cutting is how it
	// cuts them. A recording opened by this transcriber is cut by them.
	segmenter *ort.Session
	cutting   SegmenterModel

	// workers is how many goroutines share the signal processing of a stretch.
	workers int

	model TranscriptionModel

	// One set of sessions, one stretch at a time. The library is safe to call
	// from several goroutines, and a stretch is heard start to finish.
	mu sync.Mutex
}

// Open loads the models and compiles them, reading the weights. The result is
// used for any number of recordings until it is closed.
func Open(ctx context.Context, cfg Config) (*Transcriber, error) {
	opened, found, err := locateModels(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pieces, err := tokens(found.tokens)
	if err != nil {
		return nil, err
	}
	blank := pieces.getIndex(blankPiece)
	if blank < 0 {
		return nil, fmt.Errorf("%s names no %s", found.tokens, blankPiece)
	}

	out := &Transcriber{
		pieces:  pieces,
		blank:   blank,
		cutting: cfg.Segmenter,
		workers: cfg.Workers,
		model: TranscriptionModel{
			Model:     getModelName(cfg.Model.Name, found.encoder),
			Segmenter: getModelName(cfg.Segmenter.Name, found.segmenter),
			Cutting:   cfg.Segmenter.DescribeSegmentSettings(),
			From:      found.from,
		},
	}
	threads := cfg.sessionThreads()
	for _, one := range []struct {
		dst     **ort.Session
		at      string
		kind    string
		threads int
	}{
		{&out.encoder, found.encoder, "encoder", threads.encoder},
		{&out.decoder, found.decoder, "decoder", threads.decoder},
		{&out.joiner, found.joiner, "joiner", threads.joiner},
		{&out.segmenter, found.segmenter, "segmenter model", threads.segmenter},
	} {
		session, err := openSession(opened.engine, one.at, one.threads)
		if err != nil {
			out.Close()
			return nil, fmt.Errorf("the %s %s: %w", one.kind, one.at, err)
		}
		*one.dst = session
	}
	return out, nil
}

// openSession compiles the model at path with the given intra-op thread count.
func openSession(engine *ort.Engine, path string, threads int) (*ort.Session, error) {
	options, _, err := onnxruntime.NewSessionOptions(engine, onnx.SessionSettings{
		Provider: getSessionProvider(),
		Threads:  threads,
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()
	return onnxruntime.NewSession(engine, path, options)
}

// GetTranscriptionModel is what every recording this transcriber hears was heard by.
func (t *Transcriber) GetTranscriptionModel() TranscriptionModel { return t.model }

// Close lets go of the models this transcriber loaded. The runtime they ran on
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

	feature, frames := logMel(audio.Samples, t.workers)
	if frames < 2 {
		return "", nil
	}

	signal, err := ort.NewTensor([]int64{1, melBands, int64(frames)}, feature)
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

	said, err := t.decode(ctx, out)
	if err != nil {
		return "", err
	}
	return t.pieces.text(said), nil
}

// decode is what the encoder's frames say, read one at a time.
func (t *Transcriber) decode(ctx context.Context, out map[string]*ort.Value) ([]int, error) {
	return t.decodeWith(ctx, out, true)
}

// decodeWith is decode with the predictor and the joiner run through the raw
// runtime when useRaw holds and the runtime offers it, and through the library
// otherwise. The first position of each graph always runs through the library,
// which checks the size of what the graph answers.
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
		defer decoderRun.close()
	}
	if joinerRun != nil {
		defer joinerRun.close()
	}
	predicted, joined := false, false
	var scores []float32

	result, err := transducer{
		frames:  frames.width,
		blank:   t.blank,
		encoded: encoded,
		encoder: func(at int, dst []float32) []float32 {
			frames.copyFrame(at, dst)
			return dst
		},
		predictor: func(token int) ([]float32, error) {
			buf.target[0] = int32(token)
			if decoderRun == nil || !predicted {
				predicted = true
				return t.predict(decoderInputs, buf.prediction, buf.state, buf.cell)
			}
			return t.predictRaw(decoderRun, buf.prediction, buf.state, buf.cell)
		},
		joint: func(encodedFrame, _ []float32) ([]float32, error) {
			var err error
			if joinerRun == nil || !joined {
				joined = true
				scores, err = t.computeJoint(joinerInputs, buf.frame, encodedFrame, scores)
			} else {
				scores, err = t.computeJointRaw(joinerRun, buf.frame, encodedFrame, scores)
			}
			return scores, err
		},
	}.decode(ctx)

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

// readFrames reads the frames of the encoder's answer and how many carry the stretch.
func readFrames(out map[string]*ort.Value) (encodedFrames, error) {
	frames, ok := out[outputFrames]
	if !ok {
		return encodedFrames{}, fmt.Errorf("the encoder answered with %v and not %s", getNames(out), outputFrames)
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

// readFrameCount is the count of frames the encoder reports, where it reports
// one that makes sense.
func readFrameCount(counted *ort.Value) (int, bool) {
	lengths, err := ort.GetTensorData[int64](counted)
	if err != nil || len(lengths) == 0 || lengths[0] < 0 {
		return 0, false
	}
	return int(lengths[0]), true
}

// copyFrame writes the frame at a position into dst, one value from each
// channel.
func (f encodedFrames) copyFrame(at int, dst []float32) {
	for c := range dst {
		dst[c] = f.data[c*f.stride+at]
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

	targetTensor, countTensor, stateTensor, cellTensor, frameTensor, saidTensor *ort.Value
}

// newDecodeBuffers makes the buffers and the tensors over them.
func newDecodeBuffers() (*decodeBuffers, error) {
	b := &decodeBuffers{
		target:     []int32{0},
		count:      []int32{1},
		state:      make([]float32, layers*hidden),
		cell:       make([]float32, layers*hidden),
		prediction: make([]float32, hidden),
		frame:      make([]float32, encoded),
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
		{&b.saidTensor, func() (*ort.Value, error) { return ort.NewTensor([]int64{1, hidden, 1}, b.prediction) }},
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

// destroy releases the tensors that were made.
func (b *decodeBuffers) destroy() {
	for _, tensor := range []*ort.Value{
		b.targetTensor, b.countTensor, b.stateTensor, b.cellTensor, b.frameTensor, b.saidTensor,
	} {
		if tensor != nil {
			tensor.Destroy()
		}
	}
}

// keepAlive holds the buffers until the runs over them are done.
func (b *decodeBuffers) keepAlive() {
	runtime.KeepAlive(b.target)
	runtime.KeepAlive(b.count)
	runtime.KeepAlive(b.state)
	runtime.KeepAlive(b.cell)
	runtime.KeepAlive(b.prediction)
	runtime.KeepAlive(b.frame)
}

// getDecoderInputs are the tensors the predictor is run over, by name.
func (b *decodeBuffers) getDecoderInputs() map[string]*ort.Value {
	return map[string]*ort.Value{
		inputTargets: b.targetTensor,
		inputTargetN: b.countTensor,
		inputState:   b.stateTensor,
		inputCell:    b.cellTensor,
	}
}

// getJoinerInputs are the tensors the joiner is run over, by name.
func (b *decodeBuffers) getJoinerInputs() map[string]*ort.Value {
	return map[string]*ort.Value{
		inputEncoded: b.frameTensor,
		inputDecoded: b.saidTensor,
	}
}

// makeRawRuns are the predictor and the joiner run through the raw runtime, or
// nothing for each where useRaw does not hold or the runtime does not offer it.
func (t *Transcriber) makeRawRuns(b *decodeBuffers, useRaw bool) (decoderRun, joinerRun *rawRun) {
	calls, ok := getRawCalls()
	if !ok || !useRaw {
		return nil, nil
	}
	decoderRun, _ = newRawRun(calls, t.decoder, decoderInputNames, decoderOutputs,
		[]*ort.Value{b.targetTensor, b.countTensor, b.stateTensor, b.cellTensor})
	joinerRun, _ = newRawRun(calls, t.joiner, joinerInputNames, joinerOutputs,
		[]*ort.Value{b.frameTensor, b.saidTensor})
	return decoderRun, joinerRun
}

// predict is one position of the predictor: the token that was said and the
// state it leaves behind.
func (t *Transcriber) predict(inputs map[string]*ort.Value, said, state, cell []float32) ([]float32, error) {
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
		{said, outputFrames},
		{state, outputState},
		{cell, outputCell},
	} {
		if err := copyPrediction(out, one.name, one.into); err != nil {
			return nil, err
		}
	}
	return said, nil
}

// copyPrediction copies the output of the predictor with the given name into
// dst, which it must fill.
func copyPrediction(out map[string]*ort.Value, name string, dst []float32) error {
	value, ok := out[name]
	if !ok {
		return fmt.Errorf("the predictor answered with %v and not %s", getNames(out), name)
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

// predictRaw is predict run on the outputs the run owns. The three are copied
// out before the next run writes over them.
func (t *Transcriber) predictRaw(run *rawRun, prediction, state, cell []float32) ([]float32, error) {
	if err := run.run(); err != nil {
		return nil, err
	}
	for i, into := range [...][]float32{prediction, state, cell} {
		if err := run.copyOutput(i, into); err != nil {
			return nil, err
		}
	}
	return prediction, nil
}

// computeJointRaw is joint run on the output the run owns, copied into scoresDst, which
// holds as many scores as the first position answered.
func (t *Transcriber) computeJointRaw(run *rawRun, frameDst, frame, scoresDst []float32) ([]float32, error) {
	copy(frameDst, frame)
	if err := run.run(); err != nil {
		return nil, err
	}
	if err := run.copyOutput(0, scoresDst); err != nil {
		return nil, err
	}
	return scoresDst, nil
}

// computeJoint is what one frame of the encoder and one position of the predictor say
// together: a score for every token, and after them a score for every number of
// frames to step on by.
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
		return nil, fmt.Errorf("the joiner answered with %v and not %s", getNames(out), outputFrames)
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

func getNames(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
