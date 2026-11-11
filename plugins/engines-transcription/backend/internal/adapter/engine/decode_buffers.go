package engine

import (
	"fmt"
	"runtime"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/ortraw"
)

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
