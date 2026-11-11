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
