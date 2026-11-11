package engine

import (
	"fmt"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/ortraw"
)

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
