package engine

import (
	"fmt"
	"image"
	"runtime"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"
	"golang.org/x/image/draw"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

const layoutSide = 800

const (
	inputShape  = "im_shape"
	inputImage  = "image"
	inputScale  = "scale_factor"
	outputBoxes = "fetch_name_0"
)

const columns = 7

// Layout divides a page into its parts.
type Layout struct {
	session *ort.Session
	labels  []string
	minimum float32
	overlap float64

	mu    sync.Mutex
	small *image.RGBA
	flat  []float32
}

// OpenLayout loads the model that divides a page.
func OpenLayout(engine *ort.Engine, path string, options *ort.SessionOptions, labels []string, minimum float32, overlap float64) (*Layout, error) {
	if len(labels) == 0 {
		return nil, fmt.Errorf("the layout model's regions have no names")
	}
	session, err := onnx.NewSession(engine, path, options)
	if err != nil {
		return nil, fmt.Errorf("layout model %s: %w", path, err)
	}
	return &Layout{
		session: session,
		labels:  labels,
		minimum: minimum,
		overlap: overlap,
	}, nil
}

// Close releases the layout session.
func (l *Layout) Close() error {
	l.session.Destroy()
	return nil
}

// Regions divides one page and returns its parts in reading order.
func (l *Layout) Regions(page image.Image) ([]Region, error) {
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

	flat := l.getPlanes(page)
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

	answer, target := out[outputBoxes]
	if !target {
		return nil, fmt.Errorf("the layout model answered with %v and not %s", getNames(out), outputBoxes)
	}
	data, err := ort.GetTensorData[float32](answer)
	if err != nil {
		return nil, err
	}

	regions := make([]Region, 0, len(data)/columns)
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
		regions = append(regions, Region{
			Label: l.labels[class],
			Score: score,
			Rect:  rect,
			Order: int(data[i+6]),
		})
	}
	return Distinct(regions, l.overlap), nil
}

func (l *Layout) getPlanes(page image.Image) []float32 {
	if l.small == nil {
		l.small = image.NewRGBA(image.Rect(0, 0, layoutSide, layoutSide))
	}
	if len(l.flat) != 3*layoutSide*layoutSide {
		l.flat = make([]float32, 3*layoutSide*layoutSide)
	}

	draw.BiLinear.Scale(l.small, l.small.Bounds(), page, page.Bounds(), draw.Src, nil)

	plane := layoutSide * layoutSide
	pix := l.small.Pix
	flat := l.flat
	for pos := 0; pos < plane; pos++ {
		i := pos * 4
		flat[pos] = float32(pix[i]) / 255
		flat[plane+pos] = float32(pix[i+1]) / 255
		flat[2*plane+pos] = float32(pix[i+2]) / 255
	}
	return flat
}

func getNames(m map[string]*ort.Value) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
