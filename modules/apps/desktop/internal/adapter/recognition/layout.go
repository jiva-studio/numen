package recognition

import (
	"fmt"
	"image"
	"runtime"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"
	"golang.org/x/image/draw"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/ocr"
)

// The side the layout model reads a page at. Everything it answers is in the
// page's own coordinates, because the scale it was given travels with the
// request.
const layoutSide = 800

// The names the model's graph gives what it is asked and what it answers.
const (
	inputShape  = "im_shape"
	inputImage  = "image"
	inputScale  = "scale_factor"
	outputBoxes = "fetch_name_0"
)

// A row of the answer is a class, a score, a rectangle and the place the region
// takes in the order the page is read. Sorting on that last number is the whole
// of what would otherwise be a cut of the page into columns.
const columns = 7

// A Layout divides a page into its parts.
type Layout struct {
	session *ort.Session
	labels  []string
	minimum float32
	overlap float64

	// One session, one page at a time. The library is safe to call from several
	// goroutines, but a page is read start to finish and there is nothing to
	// gain from interleaving two.
	mu sync.Mutex
}

// OpenLayout loads the model that divides a page.
func OpenLayout(engine *ort.Engine, path string, options *ort.SessionOptions, labels []string, minimum float32, overlap float64) (*Layout, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("the layout model's regions have no names")
	}
	session, err := engine.NewSession(path, options)
	if err != nil {
		return nil, fmt.Errorf("layout model %s: %w", path, err)
	}
	return &Layout{session: session, labels: labels, minimum: minimum, overlap: overlap}, nil
}

func (l *Layout) Close() error {
	l.session.Destroy()
	return nil
}

// Regions divides one page and returns its parts in the order it is read.
func (l *Layout) Regions(page image.Image) ([]ocr.Region, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bounds := page.Bounds()
	if bounds.Empty() {
		return nil, nil
	}

	side := []float32{layoutSide, layoutSide}
	ratio := []float32{layoutSide / float32(bounds.Dy()), layoutSide / float32(bounds.Dx())}

	shape, err := ort.NewTensor([]int64{1, 2}, side)
	if err != nil {
		return nil, err
	}
	defer shape.Destroy()
	scale, err := ort.NewTensor([]int64{1, 2}, ratio)
	if err != nil {
		return nil, err
	}
	defer scale.Destroy()
	// The library keeps a pointer into this and nothing else does, so it is held
	// in a variable until the run is over. A slice a tensor alone refers to is a
	// slice the collector may take back, and what the model then reads is
	// whatever is there instead.
	flat := planes(page)
	pixels, err := ort.NewTensor([]int64{1, 3, layoutSide, layoutSide}, flat)
	if err != nil {
		return nil, err
	}
	defer pixels.Destroy()

	out, err := l.session.Run(map[string]*ort.Value{
		inputShape: shape, inputImage: pixels, inputScale: scale,
	})
	runtime.KeepAlive(flat)
	runtime.KeepAlive(side)
	runtime.KeepAlive(ratio)
	if err != nil {
		return nil, err
	}
	for _, v := range out {
		defer v.Destroy()
	}

	answer, held := out[outputBoxes]
	if !held {
		return nil, fmt.Errorf("the layout model answered with %v and not %s", names(out), outputBoxes)
	}
	data, err := ort.GetTensorData[float32](answer)
	if err != nil {
		return nil, err
	}

	regions := make([]ocr.Region, 0, len(data)/columns)
	for i := 0; i+columns <= len(data); i += columns {
		class, score := int(data[i]), data[i+1]
		if class < 0 || class >= len(l.labels) || score < l.minimum {
			continue
		}
		rect := image.Rect(int(data[i+2]), int(data[i+3]), int(data[i+4]), int(data[i+5])).
			Canon().Intersect(bounds)
		if rect.Empty() {
			continue
		}
		regions = append(regions, ocr.Region{
			Label: l.labels[class],
			Score: score,
			Rect:  rect,
			Order: int(data[i+6]),
		})
	}
	return ocr.Distinct(regions, l.overlap), nil
}

// planes is the page as the model asks for it: one plane a colour, each pixel a
// fraction of one. The only scaling it wants is a byte into that fraction.
func planes(page image.Image) []float32 {
	small := image.NewRGBA(image.Rect(0, 0, layoutSide, layoutSide))
	draw.CatmullRom.Scale(small, small.Bounds(), page, page.Bounds(), draw.Src, nil)

	plane := layoutSide * layoutSide
	out := make([]float32, 3*plane)
	for y := 0; y < layoutSide; y++ {
		for x := 0; x < layoutSide; x++ {
			i := small.PixOffset(x, y)
			at := y*layoutSide + x
			out[at] = float32(small.Pix[i]) / 255
			out[plane+at] = float32(small.Pix[i+1]) / 255
			out[2*plane+at] = float32(small.Pix[i+2]) / 255
		}
	}
	return out
}

func names(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
