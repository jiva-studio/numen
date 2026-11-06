package engine

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"os"
	"strings"

	read "github.com/getcharzp/go-ocr"
	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

// Model describes active OCR models.
type Model = recognition.Model

// readLines reads the lines of one part of a page.
type readLines = func(image.Image) ([]read.RecResult, error)

// Recogniser executes OCR recognition models on images.
type Recogniser struct {
	shape  *Layout
	lines  []*paddle.Engine
	layout func(image.Image) ([]Region, error)
	read   []readLines
	body   map[string]bool
	head   map[string]int
	margin int
	jobs   int
	model  Model
}

// Open loads the layout and PaddleOCR recognition models through the library.
func Open(ctx context.Context, library *onnx.Library, cfg Config) (*Recogniser, error) {
	engine, pos, found, err := locateModels(ctx, library, cfg)
	if err != nil {
		return nil, err
	}
	options, _, err := library.NewSessionOptions(engine, onnx.SessionSettings{
		Provider: onnx.GetSessionProvider(),
		Threads:  cfg.threads(),

		ComputeUnits: onnx.ComputeUnitsDefault,
		ModelPath:    found.layout,
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	layout, err := OpenLayout(engine, found.layout, options,
		cfg.Layout.labels(), cfg.Layout.minimum(), cfg.Layout.overlap())
	if err != nil {
		return nil, err
	}

	lines, err := openLines(pos, found, cfg)
	if err != nil {
		return nil, errors.Join(err, layout.Close())
	}
	reads := make([]readLines, 0, len(lines))
	names := make([]string, 0, len(lines))
	for i, line := range lines {
		reads = append(reads, readUnscaled(line.RunOCR, cfg.Detect.maxSide()))
		names = append(names, name(found.recognise[i]))
	}

	return &Recogniser{
		shape:  layout,
		lines:  lines,
		layout: layout.Regions,
		read:   reads,
		body:   toSet(cfg.Regions.body()),
		head:   getDepths(cfg.Regions.head()),
		margin: cfg.Layout.margin(),
		jobs:   cfg.jobs(),
		model: Model{
			Layout:     name(found.layout),
			Recogniser: strings.Join(names, "+"),
			DPI:        cfg.Recognise.dpi(),
			From:       found.from,
		},
	}, nil
}

// openLines opens one line engine per recognition model, each with the same detector. An engine
// that fails to open closes the ones opened before it.
func openLines(runtime string, found paths, cfg Config) ([]*paddle.Engine, error) {
	lines := make([]*paddle.Engine, 0, len(found.recognise))
	for _, model := range found.recognise {
		one, err := openLine(runtime, found.detect, model, cfg)
		if err != nil {
			for _, line := range lines {
				line.Destroy()
			}
			return nil, err
		}
		lines = append(lines, one)
	}
	return lines, nil
}

// openLine opens the line engine of one recognition model. The engine reads the dictionary written
// for it while it opens, and the dictionary is removed after.
func openLine(runtime, detect, model string, cfg Config) (*paddle.Engine, error) {
	classes, dict, err := alphabet(model, cfg.Recognise)
	if err != nil {
		return nil, err
	}
	lines, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath:  runtime,
		DetModelPath:        detect,
		RecModelPath:        model,
		DictPath:            dict,
		RecModelNumClasses:  classes,
		DetMaxSideLen:       cfg.Detect.maxSide(),
		DetOutsideExpandPix: cfg.Detect.getExpansion(),
		HeatmapThreshold:    cfg.Detect.minimum(),
		RecHeight:           cfg.Recognise.height(),
		NumThreads:          cfg.threads(),
		ThreadCount:         cfg.jobs(),
		UseCuda:             false,
	})
	removeErr := os.Remove(dict)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("the recogniser %s: %w", name(model), err), removeErr)
	}
	if removeErr != nil {
		lines.Destroy()
		return nil, removeErr
	}
	return lines, nil
}

// GetRecognitionModel returns model identity information.
func (r *Recogniser) GetRecognitionModel() Model { return r.model }

// Close releases loaded model sessions.
func (r *Recogniser) Close() error {
	for _, line := range r.lines {
		line.Destroy()
	}
	if r.shape != nil {
		return r.shape.Close()
	}
	return nil
}

// Recognise extracts layout regions and text lines from a page image.
func (r *Recogniser) Recognise(ctx context.Context, page image.Image) ([]Block, error) {
	regions, err := r.layout(page)
	if err != nil {
		return nil, err
	}

	targets := r.getTargets(page, regions)
	if len(targets) == 0 {
		return nil, nil
	}

	results := make([]outcome, len(targets))

	runBounded(ctx, len(targets), r.jobs, func(pos int) {
		found, err := r.readPart(targets[pos].crop)
		results[pos] = outcome{found: found, err: err}
	})

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var out []Block
	var hadError bool
	var failed error
	for i, res := range results {
		if res.err != nil {
			hadError, failed = true, res.err
			continue
		}
		if block, ok := r.makeBlock(targets[i], res.found); ok {
			out = append(out, block)
		}
	}
	if len(out) == 0 && hadError {
		return nil, fmt.Errorf("every part of the page: %w", failed)
	}
	return out, nil
}

type recogniseTarget struct {
	region Region
	crop   image.Image
	corner image.Point
}

type outcome struct {
	found []read.RecResult
	err   error
}

func (r *Recogniser) getTargets(page image.Image, regions []Region) []recogniseTarget {
	var targets []recogniseTarget
	for _, region := range regions {
		if !r.body[region.Label] {
			continue
		}
		crop, corner := cropRegion(page, region.Rect, r.margin)
		if crop == nil {
			continue
		}
		targets = append(targets, recogniseTarget{
			region: region,
			crop:   crop,
			corner: corner,
		})
	}
	return targets
}

func (r *Recogniser) makeBlock(t recogniseTarget, found []read.RecResult) (Block, bool) {
	lines := make([]Line, 0, len(found))
	for _, line := range found {
		lines = append(lines, Line{
			Box: image.Rect(
				line.Box[0]+t.corner.X, line.Box[1]+t.corner.Y,
				line.Box[2]+t.corner.X, line.Box[3]+t.corner.Y,
			),
			Text:  collapseSpaces(line.Text),
			Score: line.Score,
		})
	}
	text, boxes := Assemble(lines)
	if text == "" {
		// gomutants:disable-next-line RETURN_ZERO,INTEGER_DECREMENT,INTEGER_INCREMENT reason="Callers ignore the returned Block when the boolean result is false."
		return Block{Label: "", Text: "", IsHeading: false, Depth: 0, Boxes: nil}, false
	}
	depth, head := r.head[t.region.Label]
	return Block{
		Label:     t.region.Label,
		Text:      text,
		IsHeading: head,
		Depth:     depth,
		Boxes:     boxes,
	}, true
}

// cropRegion copies the region, widened by the margin, into an image whose bounds start at the origin:
// the line detector reports its boxes from the origin, and the recogniser crops them out of the image
// by its own bounds. The corner is the position of the copy on the page.
func cropRegion(page image.Image, rect image.Rectangle, margin int) (image.Image, image.Point) {
	wider := rect.Inset(-margin).Intersect(page.Bounds())
	if wider.Empty() {
		// gomutants:disable-next-line RETURN_ZERO reason="image.Point{X: 0, Y: 0} is the zero value of image.Point."
		return nil, image.Point{X: 0, Y: 0}
	}
	out := image.NewRGBA(image.Rect(0, 0, wider.Dx(), wider.Dy()))
	draw.Draw(out, out.Bounds(), page, wider.Min, draw.Src)
	return out, wider.Min
}

// readUnscaled reads a part of a page at the scale the page was rendered at. The line detector scales
// the longer side of what it is given to side, and widens each line it finds by a number of pixels
// at that scale; a part smaller than side would be enlarged, and the accents above and below its
// lines cut off. Such a part is read on a canvas side wide, filled with the colour of its corner.
func readUnscaled(run readLines, side int) readLines {
	return func(crop image.Image) ([]read.RecResult, error) {
		bounds := crop.Bounds()
		if max(bounds.Dx(), bounds.Dy()) >= side {
			return run(crop)
		}
		canvas := image.NewRGBA(image.Rect(0, 0, side, (bounds.Dy()+31)/32*32))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(crop.At(bounds.Min.X, bounds.Min.Y)), image.Point{}, draw.Src)
		draw.Draw(canvas, bounds.Sub(bounds.Min), crop, bounds.Min, draw.Src)
		found, err := run(canvas)
		for i := range found {
			box := &found[i].Box
			box[2], box[3] = min(box[2], bounds.Dx()), min(box[3], bounds.Dy())
		}
		return found, err
	}
}

func getDepths(names []string) map[string]int {
	out := make(map[string]int, len(names))
	for i, n := range names {
		out[n] = i
	}
	return out
}

func toSet(names []string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

func collapseSpaces(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// NewTestRecogniser creates a Recogniser with custom layout and read functions for testing: one
// read function per recognition model, in the order a tie is broken in.
func NewTestRecogniser(
	layout func(image.Image) ([]Region, error),
	read []func(image.Image) ([]read.RecResult, error),
	model Model,
) *Recogniser {
	return &Recogniser{
		shape:  nil,
		lines:  nil,
		layout: layout,
		read:   read,
		body:   map[string]bool{labelText: true, "paragraph": true, labelParagraphTitle: true},
		head:   map[string]int{labelParagraphTitle: 1},
		margin: 10,
		jobs:   1,
		model:  model,
	}
}
