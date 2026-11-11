package engine

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"strings"

	read "github.com/getcharzp/go-ocr"
	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine/onnx"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
)

// Model describes active OCR models.
type Model = recognition.Model

// Recognizer executes OCR recognition models on images.
type Recognizer struct {
	shape  *Layout
	lines  *paddle.Engine
	layout func(image.Image) ([]Region, error)
	read   func(image.Image) ([]read.RecResult, error)
	body   map[string]bool
	head   map[string]int
	margin int
	jobs   int
	model  Model
}

// Open loads the layout and PaddleOCR recognition models through the library.
func Open(ctx context.Context, library *onnx.Library, cfg settings.Config) (*Recognizer, error) {
	engine, pos, found, err := locateModels(ctx, library, cfg)
	if err != nil {
		return nil, err
	}
	options, _, err := library.NewSessionOptions(engine, onnx.SessionSettings{
		Provider: onnx.GetSessionProvider(),
		Threads:  getThreads(cfg),

		ComputeUnits: onnx.ComputeUnitsDefault,
		ModelPath:    found.layout,
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	layout, err := OpenLayout(engine, found.layout, options,
		cfg.Layout.GetLabels(), cfg.Layout.GetMinimum(), cfg.Layout.GetOverlap())
	if err != nil {
		return nil, err
	}

	classes, dict, err := alphabet(found.recognize, cfg.Recognize)
	if err != nil {
		return nil, errors.Join(err, layout.Close())
	}
	lines, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath:  pos,
		DetModelPath:        found.detect,
		RecModelPath:        found.recognize,
		DictPath:            dict,
		RecModelNumClasses:  classes,
		DetMaxSideLen:       cfg.Detect.GetMaxSide(),
		DetOutsideExpandPix: cfg.Detect.GetExpansion(),
		HeatmapThreshold:    cfg.Detect.GetMinimum(),
		RecHeight:           cfg.Recognize.GetHeight(),
		NumThreads:          getThreads(cfg),
		ThreadCount:         cfg.GetJobs(),
		UseCuda:             false,
	})
	if err != nil {
		return nil, errors.Join(fmt.Errorf("the recognizer: %w", err), layout.Close())
	}

	return &Recognizer{
		shape:  layout,
		lines:  lines,
		layout: layout.Regions,
		read:   lines.RunOCR,
		body:   toSet(cfg.Regions.GetBody()),
		head:   getDepths(cfg.Regions.GetHead()),
		margin: cfg.Layout.GetMargin(),
		jobs:   cfg.GetJobs(),
		model: Model{
			Layout:     name(found.layout),
			Recognizer: name(found.recognize),
			DPI:        cfg.Recognize.GetDPI(),
			From:       found.from,
		},
	}, nil
}

// GetRecognitionModel returns model identity information.
func (r *Recognizer) GetRecognitionModel() Model { return r.model }

// Close releases loaded model sessions.
func (r *Recognizer) Close() error {
	if r.lines != nil {
		r.lines.Destroy()
	}
	if r.shape != nil {
		return r.shape.Close()
	}
	return nil
}

// Recognize extracts layout regions and text lines from a page image.
func (r *Recognizer) Recognize(ctx context.Context, page image.Image) ([]Block, error) {
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
		found, err := r.read(targets[pos].crop)
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

type recognizeTarget struct {
	region Region
	crop   image.Image
	corner image.Point
}

type outcome struct {
	found []read.RecResult
	err   error
}

func (r *Recognizer) getTargets(page image.Image, regions []Region) []recognizeTarget {
	var targets []recognizeTarget
	for _, region := range regions {
		if !r.body[region.Label] {
			continue
		}
		crop, corner := cropRegion(page, region.Rect, r.margin)
		if crop == nil {
			continue
		}
		targets = append(targets, recognizeTarget{
			region: region,
			crop:   crop,
			corner: corner,
		})
	}
	return targets
}

func (r *Recognizer) makeBlock(t recognizeTarget, found []read.RecResult) (Block, bool) {
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

// NewTestRecognizer creates a Recognizer with custom layout and read functions for testing.
func NewTestRecognizer(
	layout func(image.Image) ([]Region, error),
	read func(image.Image) ([]read.RecResult, error),
	model Model,
) *Recognizer {
	return &Recognizer{
		shape:  nil,
		lines:  nil,
		layout: layout,
		read:   read,
		body:   map[string]bool{settings.LabelText: true, "paragraph": true, settings.LabelParagraphTitle: true},
		head:   map[string]int{settings.LabelParagraphTitle: 1},
		margin: 10,
		jobs:   1,
		model:  model,
	}
}
