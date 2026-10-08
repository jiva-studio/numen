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
package transcription

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
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

	model port.TranscriptionModel

	// One set of sessions, one stretch at a time. The library is safe to call
	// from several goroutines, and a stretch is heard start to finish.
	mu sync.Mutex
}

// Open loads the models and compiles them. It is expensive — the weights are
// read — and the result is reusable for the life of the process.
func Open(ctx context.Context, cfg Config) (*Transcriber, error) {
	opened, found, err := locate(ctx, cfg)
	if err != nil {
		return nil, err
	}
	options, _, err := onnxruntime.NewSessionOptions(opened.engine, onnxruntime.SessionSettings{
		Threads: cfg.threads(),
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	said, err := tokens(found.tokens)
	if err != nil {
		return nil, err
	}
	blank := said.getIndex(blankPiece)
	if blank < 0 {
		return nil, fmt.Errorf("%s names no %s", found.tokens, blankPiece)
	}

	out := &Transcriber{
		pieces:  said,
		blank:   blank,
		cutting: cfg.Segmenter,
		model: port.TranscriptionModel{
			Model:     getModelName(cfg.Model.Name, found.encoder),
			Segmenter: getModelName(cfg.Segmenter.Name, found.segmenter),
			Cutting:   cfg.Segmenter.describeCutting(),
			From:      found.from,
		},
	}
	for _, one := range []struct {
		dst  **ort.Session
		at   string
		kind string
	}{
		{&out.encoder, found.encoder, "encoder"},
		{&out.decoder, found.decoder, "decoder"},
		{&out.joiner, found.joiner, "joiner"},
		{&out.segmenter, found.segmenter, "segmenter model"},
	} {
		session, err := opened.engine.NewSession(one.at, options)
		if err != nil {
			out.Close()
			return nil, fmt.Errorf("the %s %s: %w", one.kind, one.at, err)
		}
		*one.dst = session
	}
	return out, nil
}

// Transcription is what every recording this transcriber hears was heard by.
func (t *Transcriber) Transcription() port.TranscriptionModel { return t.model }

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
func (t *Transcriber) Transcribe(ctx context.Context, audio port.Audio) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	feature, frames := logMel(audio.Samples)
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
	frames, ok := out[outputFrames]
	if !ok {
		return nil, fmt.Errorf("the encoder answered with %v and not %s", names(out), outputFrames)
	}
	shape, err := frames.GetShape()
	if err != nil {
		return nil, err
	}
	if len(shape) != 3 || shape[1] != encoded {
		return nil, fmt.Errorf("the encoder answered with frames of %v", shape)
	}
	width := int(shape[2])
	data, err := ort.GetTensorData[float32](frames)
	if err != nil {
		return nil, err
	}

	// The encoder says how many of the frames it wrote carry the stretch; the
	// rest are what the batch was padded to.
	if counted, ok := out[outputLength]; ok {
		if lengths, err := ort.GetTensorData[int64](counted); err == nil && len(lengths) > 0 {
			if n := int(lengths[0]); n >= 0 && n < width {
				width = n
			}
		}
	}

	target := []int32{0}
	count := []int32{1}
	state := make([]float32, layers*hidden)
	cell := make([]float32, layers*hidden)
	said := make([]float32, hidden)
	frame := make([]float32, encoded)
	var scores []float32

	targetTensor, err := ort.NewTensor([]int64{1, 1}, target)
	if err != nil {
		return nil, err
	}
	defer targetTensor.Destroy()

	countTensor, err := ort.NewTensor([]int64{1}, count)
	if err != nil {
		return nil, err
	}
	defer countTensor.Destroy()

	stateTensor, err := ort.NewTensor([]int64{layers, 1, hidden}, state)
	if err != nil {
		return nil, err
	}
	defer stateTensor.Destroy()

	cellTensor, err := ort.NewTensor([]int64{layers, 1, hidden}, cell)
	if err != nil {
		return nil, err
	}
	defer cellTensor.Destroy()

	frameTensor, err := ort.NewTensor([]int64{1, encoded, 1}, frame)
	if err != nil {
		return nil, err
	}
	defer frameTensor.Destroy()

	saidTensor, err := ort.NewTensor([]int64{1, hidden, 1}, said)
	if err != nil {
		return nil, err
	}
	defer saidTensor.Destroy()

	decoderInputs := map[string]*ort.Value{
		inputTargets: targetTensor,
		inputTargetN: countTensor,
		inputState:   stateTensor,
		inputCell:    cellTensor,
	}

	joinerInputs := map[string]*ort.Value{
		inputEncoded: frameTensor,
		inputDecoded: saidTensor,
	}

	result, err := transducer{
		frames:  width,
		blank:   t.blank,
		encoded: encoded,
		encoder: func(at int, dst []float32) []float32 {
			// The frames are channel after channel, each holding every frame.
			for c := range dst {
				dst[c] = data[c*int(shape[2])+at]
			}
			return dst
		},
		predictor: func(token int) ([]float32, error) {
			target[0] = int32(token)
			return t.predict(decoderInputs, said, state, cell)
		},
		joint: func(encodedFrame, _ []float32) ([]float32, error) {
			var err error
			scores, err = t.joint(joinerInputs, frame, encodedFrame, scores)
			return scores, err
		},
	}.decode(ctx)

	runtime.KeepAlive(target)
	runtime.KeepAlive(count)
	runtime.KeepAlive(state)
	runtime.KeepAlive(cell)
	runtime.KeepAlive(said)
	runtime.KeepAlive(frame)

	return result, err
}

// predict is one position of the predictor: the token that was said and the
// state it leaves behind.
func (t *Transcriber) predict(inputs map[string]*ort.Value, said, state, cell []float32) ([]float32, error) {
	out, err := t.decoder.Run(inputs)
	if err != nil {
		return nil, err
	}

	for _, one := range [...]struct {
		into []float32
		name string
	}{
		{said, outputFrames},
		{state, outputState},
		{cell, outputCell},
	} {
		value, ok := out[one.name]
		if !ok {
			for _, v := range out {
				v.Destroy()
			}
			return nil, fmt.Errorf("the predictor answered with %v and not %s", names(out), one.name)
		}
		raw, err := ort.GetTensorData[float32](value)
		if err != nil {
			for _, v := range out {
				v.Destroy()
			}
			return nil, err
		}
		copy(one.into, raw)
	}
	for _, v := range out {
		v.Destroy()
	}
	return said, nil
}

// joint is what one frame of the encoder and one position of the predictor say
// together: a score for every token, and after them a score for every number of
// frames to step on by.
func (t *Transcriber) joint(inputs map[string]*ort.Value, frameDst, frame, scoresDst []float32) ([]float32, error) {
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
		return nil, fmt.Errorf("the joiner answered with %v and not %s", names(out), outputFrames)
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

func names(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
