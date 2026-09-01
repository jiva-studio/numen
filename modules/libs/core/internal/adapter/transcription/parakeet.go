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

var _ port.Transcriber = (*Transcriber)(nil)

// A Transcriber is the models this machine hears a recording with.
type Transcriber struct {
	encoder *ort.Session
	decoder *ort.Session
	joiner  *ort.Session

	pieces pieces
	blank  int

	// speech is the model that finds the stretches, and cutting is how it cuts
	// them. A recording opened by this transcriber is cut by them.
	speech  *ort.Session
	cutting SpeechModel

	named port.Transcription

	// One set of sessions, one stretch at a time. The library is safe to call
	// from several goroutines, and a stretch is heard start to finish.
	mu sync.Mutex
}

// Open loads the models and compiles them. It is expensive — the weights are
// read — and the result is reusable for the life of the process.
func Open(ctx context.Context, cfg Config) (*Transcriber, error) {
	found, err := locate(ctx, cfg)
	if err != nil {
		return nil, err
	}
	options, err := found.engine.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	if err := options.SetIntraOpNumThreads(int32(cfg.threads())); err != nil {
		return nil, err
	}

	said, err := tokens(found.tokens)
	if err != nil {
		return nil, err
	}
	blank := said.at(blankPiece)
	if blank < 0 {
		return nil, fmt.Errorf("%s names no %s", found.tokens, blankPiece)
	}

	out := &Transcriber{
		pieces:  said,
		blank:   blank,
		cutting: cfg.Speech,
		named: port.Transcription{
			Model:     named(cfg.Model.Name, found.encoder),
			Segmenter: named(cfg.Speech.Name, found.speech),
			Threshold: cfg.Speech.threshold(),
			From:      found.from,
		},
	}
	for _, one := range []struct {
		into **ort.Session
		at   string
		what string
	}{
		{&out.encoder, found.encoder, "encoder"},
		{&out.decoder, found.decoder, "decoder"},
		{&out.joiner, found.joiner, "joiner"},
		{&out.speech, found.speech, "speech model"},
	} {
		session, err := found.engine.NewSession(one.at, options)
		if err != nil {
			out.Close()
			return nil, fmt.Errorf("the %s %s: %w", one.what, one.at, err)
		}
		*one.into = session
	}
	return out, nil
}

// Transcription is what every recording this transcriber hears was heard by.
func (t *Transcriber) Transcription() port.Transcription { return t.named }

// Close lets go of the models this transcriber loaded. The runtime they ran on
// is the process's and stays.
func (t *Transcriber) Close() error {
	for _, session := range []*ort.Session{t.encoder, t.decoder, t.joiner, t.speech} {
		if session != nil {
			session.Destroy()
		}
	}
	return nil
}

// Hear is one stretch of speech, as the words it carries.
func (t *Transcriber) Hear(ctx context.Context, audio port.Audio) (string, error) {
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
	// held until the run is over. A slice a tensor alone refers to is a slice
	// the collector may take back, and what the model then reads is whatever is
	// there instead.
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

	state := make([]float32, layers*hidden)
	cell := make([]float32, layers*hidden)
	return transducer{
		frames: width,
		blank:  t.blank,
		encoded: func(at int) []float32 {
			// The frames are channel after channel, each holding every frame.
			one := make([]float32, encoded)
			for c := range one {
				one[c] = data[c*int(shape[2])+at]
			}
			return one
		},
		predict: func(token int) ([]float32, error) {
			said, next, cells, err := t.predict(token, state, cell)
			if err != nil {
				return nil, err
			}
			state, cell = next, cells
			return said, nil
		},
		joint: t.joint,
	}.decode(ctx)
}

// predict is one position of the predictor: the token that was said and the
// state it leaves behind.
func (t *Transcriber) predict(token int, state, cell []float32) (said, next, cells []float32, err error) {
	target := []int32{int32(token)}
	count := []int32{1}
	var held []*ort.Value
	defer func() {
		for _, v := range held {
			v.Destroy()
		}
	}()

	in := map[string]*ort.Value{}
	for _, one := range []struct {
		name  string
		shape []int64
		data  any
	}{
		{inputTargets, []int64{1, 1}, target},
		{inputTargetN, []int64{1}, count},
		{inputState, []int64{layers, 1, hidden}, state},
		{inputCell, []int64{layers, 1, hidden}, cell},
	} {
		value, err := ort.NewTensor(one.shape, one.data)
		if err != nil {
			return nil, nil, nil, err
		}
		held = append(held, value)
		in[one.name] = value
	}

	out, err := t.decoder.Run(in)
	runtime.KeepAlive(target)
	runtime.KeepAlive(count)
	runtime.KeepAlive(state)
	runtime.KeepAlive(cell)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, v := range out {
		defer v.Destroy()
	}

	// What the graphs answer with lives in the library's own memory, and is
	// copied out before the answer is let go of.
	for _, one := range []struct {
		into *[]float32
		name string
	}{
		{&said, outputFrames},
		{&next, outputState},
		{&cells, outputCell},
	} {
		value, ok := out[one.name]
		if !ok {
			return nil, nil, nil, fmt.Errorf("the predictor answered with %v and not %s", names(out), one.name)
		}
		raw, err := ort.GetTensorData[float32](value)
		if err != nil {
			return nil, nil, nil, err
		}
		*one.into = append([]float32(nil), raw...)
	}
	return said, next, cells, nil
}

// joint is what one frame of the encoder and one position of the predictor say
// together: a score for every token, and after them a score for every number of
// frames to step on by.
func (t *Transcriber) joint(frame, said []float32) ([]float32, error) {
	from, err := ort.NewTensor([]int64{1, encoded, 1}, frame)
	if err != nil {
		return nil, err
	}
	defer from.Destroy()
	upto, err := ort.NewTensor([]int64{1, hidden, 1}, said)
	if err != nil {
		return nil, err
	}
	defer upto.Destroy()

	out, err := t.joiner.Run(map[string]*ort.Value{inputEncoded: from, inputDecoded: upto})
	runtime.KeepAlive(frame)
	runtime.KeepAlive(said)
	if err != nil {
		return nil, err
	}
	for _, v := range out {
		defer v.Destroy()
	}
	value, ok := out[outputFrames]
	if !ok {
		return nil, fmt.Errorf("the joiner answered with %v and not %s", names(out), outputFrames)
	}
	raw, err := ort.GetTensorData[float32](value)
	if err != nil {
		return nil, err
	}
	return append([]float32(nil), raw...), nil
}

// A transducer is the three graphs as the decoding sees them: how many frames
// there are, how to reach one, what the predictor says after a token, and what
// the two say together.
type transducer struct {
	frames  int
	blank   int
	encoded func(at int) []float32
	predict func(token int) ([]float32, error)
	joint   func(frame, said []float32) ([]float32, error)
}

// decode reads the frames from the first to the last and answers with the
// tokens they carry.
//
// The joiner names a token and how many frames it covers. A token that is not
// the blank is said and moves the predictor on; the frame is left behind either
// way, so that a stretch is always read to its end.
func (d transducer) decode(ctx context.Context) ([]int, error) {
	// The predictor opens on the blank: nothing has been said yet.
	upto, err := d.predict(d.blank)
	if err != nil {
		return nil, err
	}

	var said []int
	for at := 0; at < d.frames; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		scores, err := d.joint(d.encoded(at), upto)
		if err != nil {
			return nil, err
		}
		if len(scores) <= d.blank+1 {
			return nil, fmt.Errorf("the joiner answered with %d scores and the blank is %d", len(scores), d.blank)
		}
		token := largest(scores[:d.blank+1])
		step := largest(scores[d.blank+1:])
		if token != d.blank {
			said = append(said, token)
			if upto, err = d.predict(token); err != nil {
				return nil, err
			}
		}
		if step < 1 {
			step = 1
		}
		at += step
	}
	return said, nil
}

// largest is where the highest of a run of scores stands.
func largest(scores []float32) int {
	at := 0
	for i, v := range scores {
		if v > scores[at] {
			at = i
		}
	}
	return at
}

func names(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
