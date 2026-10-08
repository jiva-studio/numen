// Package recognition reads a scanned page with models running on this
// machine.
//
// A page goes through two of them: one divides it into its parts and says in
// what order they are read, and one reads the lines inside each part. They are
// run through ONNX Runtime, reached by name at run time.
//
// Every part is read on its own image.
package recognition

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"strings"
	"sync"

	read "github.com/getcharzp/go-ocr"
	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/modules/libs/core/internal/ocr"
	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
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
	opened, found, err := locate(ctx, cfg)
	if err != nil {
		return nil, err
	}
	options, _, err := onnxruntime.NewSessionOptions(opened.engine, onnxruntime.SessionSettings{
		Threads: cfg.Recognise.threads(),
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	layout, err := OpenLayout(opened.engine, found.layout, options,
		cfg.Layout.labels(), cfg.Layout.minimum(), cfg.Layout.overlap())
	if err != nil {
		return nil, err
	}

	classes, dict, err := alphabet(found.recognise, cfg.Recognise)
	if err != nil {
		layout.Close()
		return nil, err
	}
	lines, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath:  opened.at,
		DetModelPath:        found.detect,
		RecModelPath:        found.recognise,
		DictPath:            dict,
		RecModelNumClasses:  classes,
		DetMaxSideLen:       cfg.Detect.maxSide(),
		DetOutsideExpandPix: cfg.Detect.expand(),
		HeatmapThreshold:    cfg.Detect.minimum(),
		RecHeight:           cfg.Recognise.height(),
		NumThreads:          cfg.Recognise.threads(),
		ThreadCount:         cfg.Recognise.jobs(),
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
			Layout:     name(found.layout),
			Recogniser: name(found.recognise),
			DPI:        cfg.Recognise.dpi(),
			From:       found.from,
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

// Recognise is one page: its parts, in the order the page is read, and what
// each of them says.
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

	type target struct {
		region ocr.Region
		crop   image.Image
		corner image.Point
	}
	var targets []target
	for _, region := range regions {
		if !r.body[region.Label] {
			continue
		}
		crop, corner := cropRegion(page, region.Rect, r.margin)
		if crop == nil {
			continue
		}
		targets = append(targets, target{
			region: region,
			crop:   crop,
			corner: corner,
		})
	}
	if len(targets) == 0 {
		return nil, nil
	}

	type outcome struct {
		found []read.RecResult
		err   error
	}
	results := make([]outcome, len(targets))

	if len(targets) == 1 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found, err := r.read(targets[0].crop)
		results[0] = outcome{found: found, err: err}
	} else {
		var running sync.WaitGroup
		running.Add(len(targets))
		for i := range targets {
			go func(at int) {
				defer running.Done()
				if err := ctx.Err(); err != nil {
					results[at] = outcome{err: err}
					return
				}
				found, err := r.read(targets[at].crop)
				results[at] = outcome{found: found, err: err}
			}(i)
		}
		running.Wait()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var out []ocr.Block
	// What could not be read, and the last of it. A page whose every part was
	// refused says nothing, and a page that says nothing is a blank page to
	// everything downstream.
	var refused int
	var failed error
	for i, res := range results {
		if res.err != nil {
			// One part of a page that could not be read is one part. A page is
			// hundreds of words, and the rest of them are still what it says.
			refused, failed = refused+1, res.err
			continue
		}
		t := targets[i]
		lines := make([]ocr.Line, 0, len(res.found))
		for _, line := range res.found {
			lines = append(lines, ocr.Line{
				Box: image.Rect(
					line.Box[0]+t.corner.X, line.Box[1]+t.corner.Y,
					line.Box[2]+t.corner.X, line.Box[3]+t.corner.Y,
				),
				Text:  collapseSpaces(line.Text),
				Score: line.Score,
			})
		}
		if text, boxes := ocr.Assemble(lines); text != "" {
			depth, head := r.head[t.region.Label]
			out = append(out, ocr.Block{
				Label:     t.region.Label,
				Text:      text,
				IsHeading: head,
				Depth:     depth,
				Boxes:     boxes,
			})
		}
	}
	if len(out) == 0 && refused > 0 {
		return nil, fmt.Errorf("every part of the page: %w", failed)
	}
	return out, nil
}

// cropRegion is the part of the page a region covers, widened, and drawn into an
// image of its own.
//
// Where it was cut from comes back with it: a box the recogniser returns is
// addressed from the crop, and has to be read as a box of the page. The
// widening is because the first letter of a line sits on the boundary the
// layout model drew.
func cropRegion(page image.Image, rect image.Rectangle, margin int) (image.Image, image.Point) {
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

// collapseSpaces is one line as the recogniser wrote it, with a run of space between
// words standing as one space.
func collapseSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
