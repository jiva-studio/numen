package engine

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"strings"

	read "github.com/getcharzp/go-ocr"
	"github.com/getcharzp/go-ocr/paddle"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// Model describes active OCR models.
type Model struct {
	Layout     string
	Recogniser string
	DPI        int
	From       string
}

// Recogniser executes OCR recognition models on images.
type Recogniser struct {
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

// Open loads the layout and PaddleOCR recognition models.
func Open(ctx context.Context, cfg Config) (*Recogniser, error) {
	engine, pos, found, err := locateModels(ctx, cfg)
	if err != nil {
		return nil, err
	}
	options, _, err := onnx.NewSessionOptions(engine, onnx.SessionSettings{
		Provider: onnx.GetSessionProvider(),
		Threads:  cfg.threads(),
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

	classes, dict, err := alphabet(found.recognise, cfg.Recognise)
	if err != nil {
		layout.Close()
		return nil, err
	}
	lines, err := paddle.NewEngine(paddle.Config{
		OnnxRuntimeLibPath:  pos,
		DetModelPath:        found.detect,
		RecModelPath:        found.recognise,
		DictPath:            dict,
		RecModelNumClasses:  classes,
		DetMaxSideLen:       cfg.Detect.maxSide(),
		DetOutsideExpandPix: cfg.Detect.getExpansion(),
		HeatmapThreshold:    cfg.Detect.minimum(),
		RecHeight:           cfg.Recognise.height(),
		NumThreads:          cfg.threads(),
		ThreadCount:         cfg.jobs(),
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
		body:   toSet(cfg.Regions.body()),
		head:   getDepths(cfg.Regions.head()),
		margin: cfg.Layout.margin(),
		jobs:   cfg.jobs(),
		model: Model{
			Layout:     name(found.layout),
			Recogniser: name(found.recognise),
			DPI:        cfg.Recognise.dpi(),
			From:       found.from,
		},
	}, nil
}

// GetRecognitionModel returns model identity information.
func (r *Recogniser) GetRecognitionModel() Model { return r.model }

// Close releases loaded model sessions.
func (r *Recogniser) Close() error {
	if r.lines != nil {
		r.lines.Destroy()
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
		return Block{}, false
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
		return nil, image.Point{}
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

// NewTestRecogniser creates a Recogniser with custom layout and read functions for testing.
func NewTestRecogniser(
	layout func(image.Image) ([]Region, error),
	read func(image.Image) ([]read.RecResult, error),
	model Model,
) *Recogniser {
	return &Recogniser{
		layout: layout,
		read:   read,
		body:   map[string]bool{"text": true, "paragraph": true, "paragraph_title": true},
		head:   map[string]int{"paragraph_title": 1},
		margin: 10,
		jobs:   1,
		model:  model,
	}
}
