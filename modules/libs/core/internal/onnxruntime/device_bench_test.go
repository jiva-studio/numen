package onnxruntime_test

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport/benchenv"
)

// These benchmarks time one Run of each model session on the provider, thread
// count and compute units the NUMEN_BENCH_* variables name (see benchenv). Each
// model is opened directly and fed tensors built from the input metadata its
// own file declares, with every dynamic dimension given an explicit extent.
//
// A model is recognised by the inputs and outputs its file declares, never by
// its name. The Parakeet encoder takes audio_signal and length, the decoder
// takes targets and target_length, the joiner takes encoder_outputs and
// decoder_outputs, and the voice-activity model takes input, state and sr. A
// PaddleOCR file with image and im_shape inputs is the layout model and is not
// timed. A PaddleOCR file with one rank-4 image input is the detection model
// when it answers with a rank-4 heat map, and the recognition model when it
// answers with rank-3 scores over the characters. These match the defaults in
// recognition/config.go: Detect is PP-OCRv5_mobile_det and Recognise is
// PP-OCRv6_rec_tiny.
//
// The session is created and run warmupRuns times before the timer starts.
// Each result is ms/run and a rate in the unit the model works in.

// warmupRuns is how many runs happen before the timer starts: the first run
// compiles the graph for the shape and the following ones touch its memory.
const warmupRuns = 3

// fillTokenID is the value a token tensor holds; it is an ordinary id in every
// vocabulary a model here has, and no model's padding id.
const fillTokenID = 100

// fillPeriod is the cycle of the pattern a float tensor holds.
const fillPeriod = 97

// The workloads. Each list spans the shapes the application sends a model.
const (
	vadWindow      = 512   // samples one voice-activity run reads
	vadSampleRate  = 16000 // the rate the voice-activity model is told
	detLongSide    = 1600  // the longest side of a page the detector reads (DetectModel default)
	detShortSide   = 1152  // the A4 ratio of detLongSide, a multiple of the detector's stride
	recHeight      = 48    // the height a line is scaled to (RecogniserModel default)
	joinerSteps    = 1     // encoder frames and predictor positions in one joiner run
	decoderTargets = 1     // targets in one decoder run
)

var (
	// encoderFrames is mel frames per encoder run, from a short utterance to a long segment.
	encoderFrames = []int64{100, 400, 1000}
	// recWidths is the width of a line crop in pixels.
	recWidths = []int64{160, 480, 960}
	// recBatches is how many crops one recognition run takes.
	recBatches = []int64{1, 8, 32}
	// embedBuckets is the padded token length of one embedding run.
	embedBuckets = []int64{64, 128, 256, 512}
	// embedBatches is how many texts one embedding run takes.
	embedBatches = []int64{1, 8, 32}
)

// A modelRole is the part a model plays, read off its signature.
type modelRole string

const (
	roleEncoder   modelRole = "parakeet encoder"
	roleDecoder   modelRole = "parakeet decoder"
	roleJoiner    modelRole = "parakeet joiner"
	roleVAD       modelRole = "silero vad"
	roleLayout    modelRole = "ocr layout"
	roleDetect    modelRole = "ocr detection"
	roleRecognise modelRole = "ocr recognition"
	roleOther     modelRole = "other"
)

// classifyModel names the part a model plays from the inputs and outputs it
// declares.
func classifyModel(sig modelSignature) modelRole {
	has := func(names ...string) bool {
		for _, name := range names {
			if !slices.ContainsFunc(sig.Inputs, func(in tensorSignature) bool { return in.Name == name }) {
				return false
			}
		}
		return true
	}
	switch {
	case has("audio_signal", "length"):
		return roleEncoder
	case has("targets", "target_length"):
		return roleDecoder
	case has("encoder_outputs", "decoder_outputs"):
		return roleJoiner
	case has("input", "state", "sr"):
		return roleVAD
	case has("image", "im_shape"):
		return roleLayout
	case len(sig.Inputs) == 1 && len(sig.Inputs[0].Dims) == 4 && len(sig.Outputs) == 1:
		switch len(sig.Outputs[0].Dims) {
		case 4:
			return roleDetect
		case 3:
			return roleRecognise
		}
	}
	return roleOther
}

// shapeOf is the extents of a tensor: a fixed dimension keeps the extent the
// model declares, a dynamic one takes the extent axes gives its position, and
// else the one extent rest gives every other. It refuses a dynamic dimension
// left without an extent.
func shapeOf(in tensorSignature, axes map[int]int64, rest int64) ([]int64, error) {
	dims := slices.Clone(in.Dims)
	for axis, extent := range dims {
		if extent != dynamicDim {
			continue
		}
		switch want, ok := axes[axis]; {
		case ok:
			dims[axis] = want
		case rest > 0:
			dims[axis] = rest
		default:
			return nil, fmt.Errorf("input %s: dimension %d (%q) has no extent", in.Name, axis, in.Params[axis])
		}
	}
	return dims, nil
}

// getElementCount is how many elements a tensor of these extents holds; a
// scalar holds one.
func getElementCount(dims []int64) int {
	count := 1
	for _, extent := range dims {
		count *= int(extent)
	}
	return count
}

// A workload is one shape a model is timed at.
type workload struct {
	name string
	// shape gives the extents of an input.
	shape func(in tensorSignature) ([]int64, error)
	// ints gives an integer input the value it is filled with, by name.
	ints map[string]int64
	// amount is how much a run works through, in unit.
	amount float64
	unit   string
}

func byAxes(axes map[int]int64, rest int64) func(tensorSignature) ([]int64, error) {
	return func(in tensorSignature) ([]int64, error) { return shapeOf(in, axes, rest) }
}

// fill returns the data of an input: a float tensor holds a repeating ramp, a
// mask holds ones, and any other integer tensor holds the value ints gives its
// name, else fillTokenID.
func fill(in tensorSignature, count int, ints map[string]int64) (any, error) {
	integer := func() int64 {
		if value, ok := ints[in.Name]; ok {
			return value
		}
		if strings.Contains(in.Name, "mask") {
			return 1
		}
		return fillTokenID
	}
	switch in.Type {
	case elementFloat32:
		data := make([]float32, count)
		for i := range data {
			data[i] = float32(i%fillPeriod)/fillPeriod - 0.5
		}
		return data, nil
	case elementInt64:
		data := make([]int64, count)
		for i := range data {
			data[i] = integer()
		}
		return data, nil
	case elementInt32:
		data := make([]int32, count)
		for i := range data {
			data[i] = int32(integer())
		}
		return data, nil
	case elementBool:
		data := make([]bool, count)
		for i := range data {
			data[i] = true
		}
		return data, nil
	}
	return nil, fmt.Errorf("input %s: element type %d is not one this benchmark fills", in.Name, in.Type)
}

// An inputData is the extents and the contents of one input.
type inputData struct {
	dims []int64
	data any
}

// buildInputData returns the extents and the contents of every input of the
// model at one workload.
func buildInputData(sig modelSignature, w workload) (map[string]inputData, error) {
	built := make(map[string]inputData, len(sig.Inputs))
	for _, in := range sig.Inputs {
		dims, err := w.shape(in)
		if err != nil {
			return nil, err
		}
		data, err := fill(in, getElementCount(dims), w.ints)
		if err != nil {
			return nil, err
		}
		built[in.Name] = inputData{dims: dims, data: data}
	}
	return built, nil
}

// newInputs builds the tensors of one workload.
func newInputs(b *testing.B, sig modelSignature, w workload) map[string]*ort.Value {
	b.Helper()
	built, err := buildInputData(sig, w)
	if err != nil {
		b.Fatal(err)
	}
	inputs := make(map[string]*ort.Value, len(built))
	for name, in := range built {
		value, err := ort.NewTensor(in.dims, in.data)
		if err != nil {
			b.Fatalf("input %s %v: %v", name, in.dims, err)
		}
		b.Cleanup(func() {
			value.Destroy()
			runtime.KeepAlive(in.data)
		})
		inputs[name] = value
	}
	return inputs
}

// A deviceModel is a model file and the part it plays.
type deviceModel struct {
	name string
	path string
	sig  modelSignature
}

// listModels returns the models of a role in the models directory, and skips
// the benchmark when there is none.
func listModels(b *testing.B, cfg benchenv.Config, role modelRole) []deviceModel {
	b.Helper()
	var found []deviceModel
	for _, path := range cfg.ListModels(b) {
		sig, err := readModelSignature(path)
		if err != nil {
			b.Logf("skipping %s: %v", path, err)
			continue
		}
		if classifyModel(sig) == role {
			found = append(found, deviceModel{name: benchenv.GetModelName(path), path: path, sig: sig})
		}
	}
	if len(found) == 0 {
		b.Skipf("no %s model in %s: set %s to a directory that holds one", role, cfg.Models, benchenv.ModelsVar)
	}
	return found
}

// openSession creates the session of a model on the configured device, and
// skips the benchmark when the device is not available here.
func openSession(b *testing.B, cfg benchenv.Config, path string) *ort.Session {
	b.Helper()
	engine, _, err := onnxruntime.Open(b.Context(), onnxruntime.Settings{
		Section: "indexing.recognition",
		Runtime: cfg.Runtime,
	})
	if err != nil {
		b.Skipf("onnx runtime not available: %v", err)
	}
	options, used, err := onnxruntime.NewSessionOptions(engine, cfg.Session)
	if err != nil {
		b.Skipf("provider %q is not available here: %v", cfg.Session.Provider, err)
	}
	defer options.Destroy()
	if cfg.Session.Provider != onnxruntime.ProviderAuto && cfg.Session.Provider != used {
		b.Skipf("provider %q ran as %q", cfg.Session.Provider, used)
	}
	session, err := onnxruntime.NewSession(engine, path, options)
	if err != nil {
		b.Fatalf("open %s: %v", path, err)
	}
	b.Cleanup(session.Destroy)
	b.Logf("%s on %s, %d threads, compute units %q", path, used, cfg.Session.Threads, cfg.Session.ComputeUnits)
	return session
}

func runOnce(b *testing.B, session *ort.Session, inputs map[string]*ort.Value) {
	b.Helper()
	outputs, err := session.Run(inputs)
	if err != nil {
		b.Fatal(err)
	}
	for _, value := range outputs {
		value.Destroy()
	}
}

// timeWorkloads times Run at each workload on a session that is already open.
func timeWorkloads(b *testing.B, session *ort.Session, sig modelSignature, workloads []workload) {
	for _, w := range workloads {
		b.Run(w.name, func(b *testing.B) {
			inputs := newInputs(b, sig, w)
			for range warmupRuns {
				runOnce(b, session, inputs)
			}
			for b.Loop() {
				runOnce(b, session, inputs)
			}
			elapsed := b.Elapsed()
			b.ReportMetric(float64(elapsed.Nanoseconds())/float64(b.N)/1e6, "ms/run")
			b.ReportMetric(w.amount*float64(b.N)/elapsed.Seconds(), w.unit)
		})
	}
}

// timeModels opens each model of a role and times the workloads on it, under
// the model's name.
func timeModels(b *testing.B, role modelRole, workloads func() []workload) {
	cfg := benchenv.Read(b)
	for _, model := range listModels(b, cfg, role) {
		b.Run(model.name, func(b *testing.B) {
			session := openSession(b, cfg, model.path)
			timeWorkloads(b, session, model.sig, workloads())
		})
	}
}

func BenchmarkDeviceParakeetEncoder(b *testing.B) {
	timeModels(b, roleEncoder, func() []workload {
		var out []workload
		for _, frames := range encoderFrames {
			out = append(out, workload{
				name:   fmt.Sprintf("frames=%d", frames),
				shape:  byAxes(map[int]int64{0: 1, 2: frames}, 0),
				ints:   map[string]int64{"length": frames},
				amount: float64(frames),
				unit:   "frames/s",
			})
		}
		return out
	})
}

func BenchmarkDeviceParakeetDecoder(b *testing.B) {
	timeModels(b, roleDecoder, func() []workload {
		return []workload{{
			name:   "step",
			shape:  byAxes(nil, 1),
			ints:   map[string]int64{"target_length": decoderTargets},
			amount: decoderTargets,
			unit:   "steps/s",
		}}
	})
}

func BenchmarkDeviceParakeetJoiner(b *testing.B) {
	timeModels(b, roleJoiner, func() []workload {
		return []workload{{
			name:   "step",
			shape:  byAxes(nil, joinerSteps),
			amount: joinerSteps,
			unit:   "steps/s",
		}}
	})
}

func BenchmarkDeviceSileroVAD(b *testing.B) {
	timeModels(b, roleVAD, func() []workload {
		return []workload{{
			name: "window",
			shape: func(in tensorSignature) ([]int64, error) {
				if in.Name == "input" {
					return shapeOf(in, map[int]int64{0: 1, 1: vadWindow}, 0)
				}
				return shapeOf(in, nil, 1)
			},
			ints:   map[string]int64{"sr": vadSampleRate},
			amount: 1,
			unit:   "windows/s",
		}}
	})
}

func BenchmarkDeviceOCRDetect(b *testing.B) {
	timeModels(b, roleDetect, func() []workload {
		return []workload{{
			name:   "page",
			shape:  byAxes(map[int]int64{0: 1, 2: detLongSide, 3: detShortSide}, 0),
			amount: float64(detLongSide*detShortSide) / 1e6,
			unit:   "Mpixel/s",
		}}
	})
}

func BenchmarkDeviceOCRRecognise(b *testing.B) {
	timeModels(b, roleRecognise, func() []workload {
		var out []workload
		for _, batch := range recBatches {
			for _, width := range recWidths {
				out = append(out, workload{
					name:   fmt.Sprintf("batch=%d/width=%d", batch, width),
					shape:  byAxes(map[int]int64{0: batch, 2: recHeight, 3: width}, 0),
					amount: float64(batch),
					unit:   "crops/s",
				})
			}
		}
		return out
	})
}

// getEmbedderWorkloads returns one workload per padded length and batch size.
// Every text is one segment of real tokens with no padding: input_ids holds an
// ordinary in-vocabulary id, attention_mask holds ones, and token_type_ids
// holds zeros, the only row every token-type table has.
func getEmbedderWorkloads() []workload {
	var workloads []workload
	for _, bucket := range embedBuckets {
		for _, batch := range embedBatches {
			workloads = append(workloads, workload{
				name:   fmt.Sprintf("bucket=%d/batch=%d", bucket, batch),
				shape:  byAxes(map[int]int64{0: batch, 1: bucket}, 0),
				ints:   map[string]int64{"token_type_ids": 0},
				amount: float64(batch),
				unit:   "texts/s",
			})
		}
	}
	return workloads
}

func BenchmarkDeviceEmbedder(b *testing.B) {
	cfg := benchenv.Read(b)
	path := cfg.GetEmbeddingModel(b)
	sig, err := readModelSignature(path)
	if err != nil {
		b.Fatal(err)
	}
	session := openSession(b, cfg, path)
	timeWorkloads(b, session, sig, getEmbedderWorkloads())
}
