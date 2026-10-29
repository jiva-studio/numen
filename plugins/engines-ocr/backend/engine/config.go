package engine

import (
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

// Config holds settings for recognising scanned pages.
type Config struct {
	Runtime        string                               `json:"runtime"`
	Dir            string                               `json:"dir"`
	ShouldDownload bool                                 `json:"download"`
	Layout         LayoutModel                          `json:"layout"`
	Detect         DetectModel                          `json:"detect"`
	Recognise      RecogniserModel                      `json:"recognise"`
	Regions        RegionKinds                          `json:"regions"`
	Compute        Compute                              `json:"-"`
	Progress       func(what string, done, total int64) `json:"-"`
}

// LayoutModel divides a page into its parts and puts them in reading order.
type LayoutModel struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Labels  []string `json:"labels"`
	Minimum float32  `json:"minimum"`
	Overlap float64  `json:"overlap"`
	Margin  int      `json:"margin"`
}

// DetectModel finds the lines one part holds.
type DetectModel struct {
	Name    string  `json:"name"`
	Path    string  `json:"path"`
	MaxSide int     `json:"max_side"`
	Expand  int     `json:"expand"`
	Minimum float32 `json:"minimum"`
}

// RecogniserModel configures the model that recognizes the text of a line.
type RecogniserModel struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Dict    string `json:"dict"`
	Classes int64  `json:"classes"`
	DPI     int    `json:"dpi"`
	Height  int    `json:"height"`
	Jobs    int    `json:"jobs"`
	Threads int    `json:"threads"`
}

// RegionKinds classifies page regions as body or head.
type RegionKinds struct {
	Body []string `json:"body"`
	Head []string `json:"head"`
}

// Defaults returns default recognition configuration using remote models.
func Defaults() Config {
	return Config{
		Layout: LayoutModel{
			Name: layoutDefault,
		},
		Detect: DetectModel{
			Name: detectDefault,
		},
		Recognise: RecogniserModel{
			Name: recogDefault,
			DPI:  300,
		},
		ShouldDownload: true,
	}
}

func (l LayoutModel) minimum() float32 {
	if l.Minimum <= 0 {
		return 0.4
	}
	return l.Minimum
}

func (l LayoutModel) overlap() float64 {
	if l.Overlap <= 0 {
		return 0.6
	}
	return l.Overlap
}

func (l LayoutModel) margin() int {
	if l.Margin <= 0 {
		return 10
	}
	return l.Margin
}

func (l LayoutModel) labels() []string {
	if len(l.Labels) > 0 {
		return l.Labels
	}
	return []string{
		"abstract", "algorithm", "aside_text", "chart", "content",
		"display_formula", "doc_title", "figure_title", "footer",
		"footer_image", "footnote", "formula_number", "header",
		"header_image", "image", "inline_formula", "number",
		"paragraph_title", "reference", "reference_content", "seal",
		"table", "text", "vertical_text", "vision_footnote",
	}
}

func (d DetectModel) maxSide() int {
	if d.MaxSide <= 0 {
		return 1600
	}
	return d.MaxSide
}

func (d DetectModel) getExpansion() int {
	if d.Expand <= 0 {
		return 18
	}
	return d.Expand
}

func (d DetectModel) minimum() float32 {
	if d.Minimum <= 0 {
		return 0.3
	}
	return d.Minimum
}

func (r RecogniserModel) dpi() int {
	if r.DPI <= 0 {
		return 300
	}
	return r.DPI
}

func (r RecogniserModel) height() int {
	if r.Height <= 0 {
		return 48
	}
	return r.Height
}

func (r RegionKinds) body() []string {
	if len(r.Body) > 0 {
		return r.Body
	}
	return []string{
		"text", "paragraph_title", "doc_title", "abstract", "content",
		"reference", "reference_content", "footnote", "figure_title",
	}
}

func (r RegionKinds) head() []string {
	if len(r.Head) > 0 {
		return r.Head
	}
	return []string{"doc_title", "paragraph_title"}
}

// Compute configures thread counts and session concurrency.
type Compute struct {
	Threads int
	Jobs    int
}

const (
	fallbackThreads = 4
	fallbackJobs    = 4
)

func (c Config) jobs() int {
	switch {
	case c.Recognise.Jobs > 0:
		return c.Recognise.Jobs
	case c.Compute.Jobs > 0:
		return c.Compute.Jobs
	}
	return fallbackJobs
}

func (c Config) threads() int {
	switch {
	case c.Recognise.Threads > 0:
		return c.Recognise.Threads
	case c.Compute.Threads > 0:
		return c.Compute.Threads
	}
	return min(fallbackThreads, onnx.GetDefaultThreads())
}
