// Package recognition reads a scanned page with models running on this
// machine.
//
// A page goes through two of them: one divides it into its parts and says in
// what order they are read, and one reads the lines inside each part. They are
// run through ONNX Runtime, reached by name at run time, so this builds with
// CGO_ENABLED=0 and cross-compiles from any machine to any other.
//
// Every part is read on its own image. Reading a whole page at once costs one
// detection instead of ten, and a line found that way reaches across the gutter
// of a two-column page and carries the other column's words with it.
package recognition

import (
	"context"
	"fmt"
	"image"
	"image/draw"

	read "github.com/getcharzp/go-ocr"
	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/modules/libs/core/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A Recogniser is the models this machine reads a page with.
type Recogniser struct {
	shape *Layout
	lines *paddle.Engine

	// layout is the parts of a page in the order they are read, and read is
	// what one of those parts says. A test puts its own in.
	layout func(image.Image) ([]ocr.Region, error)
	read   func(image.Image) ([]read.RecResult, error)

	body   map[string]bool
	head   map[string]int
	margin int
	model  port.RecognitionModel
}

// Open loads the models and compiles them. It is expensive — the weights are
// read — and the result is reusable for the life of the process.
func Open(ctx context.Context, cfg Config) (*Recogniser, error) {
	paths, err := locate(ctx, cfg)
	if err != nil {
		return nil, err
	}
	options, err := paths.engine.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	if err := options.SetIntraOpNumThreads(int32(cfg.Recognise.threads())); err != nil {
		return nil, err
	}

	layout, err := OpenLayout(paths.engine, paths.layout, options,
		cfg.Layout.labels(), cfg.Layout.minimum(), cfg.Layout.overlap())
	if err != nil {
		return nil, err
	}

	classes, dict, err := alphabet(paths.recognise, cfg.Recognise)
	if err != nil {
		layout.Close()
		return nil, err
	}
	lines, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath:  paths.runtime,
		DetModelPath:        paths.detect,
		RecModelPath:        paths.recognise,
		DictPath:            dict,
		RecModelNumClasses:  classes,
		DetMaxSideLen:       cfg.Detect.maxSide(),
		DetOutsideExpandPix: cfg.Detect.expand(),
		HeatmapThreshold:    cfg.Detect.minimum(),
		RecHeight:           cfg.Recognise.height(),
		NumThreads:          cfg.Recognise.threads(),
		ThreadCount:         cfg.Recognise.sessions(),
	})
	if err != nil {
		layout.Close()
		return nil, fmt.Errorf("the recogniser: %w", err)
	}

	return &Recogniser{
		shape:  layout,
		lines:  lines,
		layout: layout.Regions,
		read:   lines.RunOCR,
		body:   set(cfg.Regions.body()),
		head:   depths(cfg.Regions.head()),
		margin: cfg.Layout.margin(),
		model: port.RecognitionModel{
			Layout:     name(paths.layout),
			Recogniser: name(paths.recognise),
			DPI:        cfg.Recognise.dpi(),
			From:       paths.from,
		},
	}, nil
}

func (r *Recogniser) Recognition() port.RecognitionModel { return r.model }

// Close lets go of the models this reading loaded. The runtime they ran on is
// the process's and stays.
func (r *Recogniser) Close() error {
	r.lines.Destroy()
	return r.shape.Close()
}

// Read is one page: its parts, in the order the page is read, and what each of
// them says.
//
// A part whose kind the configuration does not ask for is not read at all. A
// running head and a page ornament are printed on every page and are not what
// the page says, and reading them costs as much as reading a paragraph.
//
// A part the configuration calls a head opens a part of the document, and
// carries how deep that part sits.
func (r *Recogniser) Recognise(ctx context.Context, page image.Image) ([]ocr.Block, error) {
	regions, err := r.layout(page)
	if err != nil {
		return nil, err
	}

	var out []ocr.Block
	// What could not be read, and the last of it. A page whose every part was
	// refused says nothing, and a page that says nothing is a blank page to
	// everything downstream.
	var refused int
	var failed error
	for _, region := range regions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !r.body[region.Label] {
			continue
		}
		crop, corner := cropped(page, region.Rect, r.margin)
		if crop == nil {
			continue
		}
		found, err := r.read(crop)
		if err != nil {
			// One part of a page that could not be read is one part. A page is
			// hundreds of words, and the rest of them are still what it says.
			refused, failed = refused+1, err
			continue
		}
		lines := make([]ocr.Line, 0, len(found))
		for _, line := range found {
			lines = append(lines, ocr.Line{
				Box: image.Rect(
					line.Box[0]+corner.X, line.Box[1]+corner.Y,
					line.Box[2]+corner.X, line.Box[3]+corner.Y,
				),
				Text:  spaced(line.Text),
				Score: line.Score,
			})
		}
		if text, stretches := ocr.Assemble(lines); text != "" {
			depth, head := r.head[region.Label]
			out = append(out, ocr.Block{
				Label:     region.Label,
				Text:      text,
				Heading:   head,
				Depth:     depth,
				Stretches: stretches,
			})
		}
	}
	if len(out) == 0 && refused > 0 {
		return nil, fmt.Errorf("every part of the page: %w", failed)
	}
	return out, nil
}

// cropped is the part of the page a region covers, widened, and drawn into an
// image of its own.
//
// Where it was cut from comes back with it: a box the recogniser returns is
// addressed from the crop, and has to be read as a box of the page. The
// widening is because the first letter of a line sits on the boundary the
// layout model drew.
func cropped(page image.Image, rect image.Rectangle, margin int) (image.Image, image.Point) {
	wider := rect.Inset(-margin).Intersect(page.Bounds())
	if wider.Empty() {
		return nil, image.Point{}
	}
	out := image.NewRGBA(image.Rect(0, 0, wider.Dx(), wider.Dy()))
	draw.Draw(out, out.Bounds(), page, wider.Min, draw.Src)
	return out, wider.Min
}

// depths is how deep the part each name opens sits: where it stands in the
// list, outermost first.
func depths(names []string) map[string]int {
	out := make(map[string]int, len(names))
	for i, n := range names {
		out[n] = i
	}
	return out
}

func set(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}
