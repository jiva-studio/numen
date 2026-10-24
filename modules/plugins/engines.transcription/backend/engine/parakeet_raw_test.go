package engine

import (
	"math"
	"os"
	"runtime"
	"slices"
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/plugins/engines.transcription/backend/engine/ortraw"
)

// openCachedTranscriber is the models a machine holds, in NUMEN_MODELS or in the
// cache. A machine without them skips and the rest of the package is still
// tested.
func openCachedTranscriber(t testing.TB) *Transcriber {
	t.Helper()
	cfg := Defaults()
	cfg.ShouldDownload = false
	if dir := os.Getenv("NUMEN_MODELS"); dir != "" {
		cfg.Dir = dir
	}
	delta, err := Open(t.Context(), cfg)
	if err != nil {
		t.Skipf("parakeet models not available: %v", err)
	}
	t.Cleanup(func() { delta.Close() })
	return delta
}

// makeFrames is a run of encoder frames that is the same on every call and
// makes the joiner return tokens.
func makeFrames(t testing.TB, width int) map[string]*ort.Value {
	data := make([]float32, encoded*width)
	seed := uint32(1)
	for i := range data {
		seed = seed*1664525 + 1013904223
		data[i] = float32(math.Sin(float64(seed>>8)/1e3)) * 2
	}
	tensor, err := ort.NewTensor([]int64{1, encoded, int64(width)}, data)
	if err != nil {
		t.Fatal(err)
	}
	// The tensor points into data and nothing else keeps it.
	t.Cleanup(func() { tensor.Destroy(); runtime.KeepAlive(data) })
	return map[string]*ort.Value{outputFrames: tensor}
}

func TestDecodingOnTheRawRuntimeReturnsTheSameTokensAsDecodingThroughTheLibrary(t *testing.T) {
	delta := openCachedTranscriber(t)

	for _, width := range []int{1, 2, 40, 120} {
		frames := makeFrames(t, width)

		want, err := delta.decodeWith(t.Context(), frames, false)
		if err != nil {
			t.Fatal(err)
		}
		got, err := delta.decodeWith(t.Context(), frames, true)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%d frames: raw run returned %v, library returned %v", width, got, want)
		}
		if width == 120 && len(want) == 0 {
			t.Error("the frames made the joiner say nothing, so the comparison holds nothing")
		}
	}
}

func TestDecodingOnTheRawRuntimeAllocatesLessThanDecodingThroughTheLibrary(t *testing.T) {
	delta := openCachedTranscriber(t)
	const short, long = 30, 130
	shortFrames, longFrames := makeFrames(t, short), makeFrames(t, long)

	perFrame := func(useRaw bool) float64 {
		measure := func(frames map[string]*ort.Value) float64 {
			return testing.AllocsPerRun(3, func() {
				if _, err := delta.decodeWith(t.Context(), frames, useRaw); err != nil {
					t.Fatal(err)
				}
			})
		}
		return (measure(longFrames) - measure(shortFrames)) / (long - short)
	}

	raw, library := perFrame(true), perFrame(false)
	if raw > library/2 {
		t.Errorf("a frame costs %.1f allocations on the raw runtime and %.1f through the library", raw, library)
	}
	if raw > 16 {
		t.Errorf("a frame costs %.1f allocations on the raw runtime", raw)
	}
}

func TestDecodingOnTheRawRuntimeReleasesEveryOutputItCreated(t *testing.T) {
	delta := openCachedTranscriber(t)
	frames := makeFrames(t, 20)

	var released int
	t.Cleanup(ortraw.ObserveRelease(func(uintptr) { released++ }))

	if _, err := delta.decodeWith(t.Context(), frames, true); err != nil {
		t.Fatal(err)
	}
	// The predictor answers three values and the joiner one.
	if want := len(decoderOutputs) + len(joinerOutputs); released != want {
		t.Errorf("released %d outputs, want %d", released, want)
	}
}

// A position is one set of the predictor's and the joiner's inputs.
type position struct {
	target, count      []int32
	state, cell        []float32
	prediction, frame  []float32
	decoderInputs      map[string]*ort.Value
	joinerInputs       map[string]*ort.Value
	decoderRun, joiner *ortraw.Graph
}

func openPosition(t *testing.T, delta *Transcriber) *position {
	t.Helper()
	p := &position{
		target: []int32{0}, count: []int32{1},
		state: make([]float32, layers*hidden), cell: make([]float32, layers*hidden),
		prediction: make([]float32, hidden), frame: make([]float32, encoded),
	}
	make1 := func(shape []int64, data any) *ort.Value {
		value, err := ort.NewTensor(shape, data)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(value.Destroy)
		return value
	}
	targetT := make1([]int64{1, 1}, p.target)
	countT := make1([]int64{1}, p.count)
	stateT := make1([]int64{layers, 1, hidden}, p.state)
	cellT := make1([]int64{layers, 1, hidden}, p.cell)
	frameT := make1([]int64{1, encoded, 1}, p.frame)
	predictionT := make1([]int64{1, hidden, 1}, p.prediction)
	p.decoderInputs = map[string]*ort.Value{inputTargets: targetT, inputTargetN: countT, inputState: stateT, inputCell: cellT}
	p.joinerInputs = map[string]*ort.Value{inputEncoded: frameT, inputDecoded: predictionT}

	calls, ok := ortraw.GetCalls()
	if !ok {
		t.Skip("the raw runtime is not registered")
	}
	p.decoderRun, _ = ortraw.NewGraph(calls, delta.decoder, decoderInputNames, decoderOutputs, []*ort.Value{targetT, countT, stateT, cellT})
	p.joiner, _ = ortraw.NewGraph(calls, delta.joiner, joinerInputNames, joinerOutputs, []*ort.Value{frameT, predictionT})
	if p.decoderRun == nil || p.joiner == nil {
		t.Skip("the sessions hold no handle to run by")
	}
	t.Cleanup(p.decoderRun.Close)
	t.Cleanup(p.joiner.Close)
	return p
}

func TestEveryPositionOnTheRawRuntimeAnswersBitForBitWhatTheLibraryAnswers(t *testing.T) {
	delta := openCachedTranscriber(t)
	library, raw := openPosition(t, delta), openPosition(t, delta)
	frames := makeFrames(t, 1)
	data, err := ort.GetTensorData[float32](frames[outputFrames])
	if err != nil {
		t.Fatal(err)
	}

	var libraryScores, rawScores []float32
	for step, token := range []int{delta.blank, 3, 17, 250, 9, 1024, 4, 4, 77, 600, 12, 5} {
		libraryScores, rawScores = compareStepInTest(t, delta, library, raw, step, token, data, libraryScores, rawScores)
	}
}

func compareStepInTest(
	t *testing.T,
	delta *Transcriber,
	library, raw *position,
	step, token int,
	data []float32,
	libraryScores, rawScores []float32,
) ([]float32, []float32) {
	t.Helper()
	library.target[0], raw.target[0] = int32(token), int32(token)
	if _, err := delta.predict(library.decoderInputs, library.prediction, library.state, library.cell); err != nil {
		t.Fatal(err)
	}
	if step == 0 {
		if _, err := delta.predict(raw.decoderInputs, raw.prediction, raw.state, raw.cell); err != nil {
			t.Fatal(err)
		}
	} else if _, err := delta.predictRaw(raw.decoderRun, raw.prediction, raw.state, raw.cell); err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2][]float32{
		"prediction": {library.prediction, raw.prediction}, "state": {library.state, raw.state}, "cell": {library.cell, raw.cell},
	} {
		if !slices.Equal(pair[0], pair[1]) {
			t.Fatalf("step %d: the raw runtime left a different %s", step, name)
		}
	}

	var err error
	if libraryScores, err = delta.computeJoint(library.joinerInputs, library.frame, data[:encoded], libraryScores); err != nil {
		t.Fatal(err)
	}
	if step == 0 {
		rawScores, err = delta.computeJoint(raw.joinerInputs, raw.frame, data[:encoded], rawScores)
	} else {
		rawScores, err = delta.computeJointRaw(raw.joiner, raw.frame, data[:encoded], rawScores)
	}
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(libraryScores, rawScores) {
		t.Fatalf("step %d: the raw runtime scored differently", step)
	}
	return libraryScores, rawScores
}
