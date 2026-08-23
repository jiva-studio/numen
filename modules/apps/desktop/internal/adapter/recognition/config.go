package recognition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// Config is what a person may change about reading a scanned page: which
// models, where they came from, how the page is read, and which of a page's
// parts are wanted.
//
// A zero field takes its default, so a settings file naming one thing leaves
// the rest alone.
type Config struct {
	// Runtime is the ONNX Runtime shared library. Empty means the one beside
	// the application, and then the one the platform holds.
	Runtime string `json:"runtime"`

	// Dir is a folder holding the models. Empty means the folder beside the
	// application, and then the download cache.
	Dir string `json:"dir"`

	// Download allows fetching what is not on this machine.
	Download bool `json:"download"`

	Layout    LayoutModel     `json:"layout"`
	Detect    DetectModel     `json:"detect"`
	Recognise RecogniserModel `json:"recognise"`
	Page      PageReading     `json:"page"`
	Regions   RegionKinds     `json:"regions"`

	// Fetching is told how far a download has got, when anything is listening.
	// It is not a setting and is not written down: it is how the wait reaches
	// whoever is watching it.
	Fetching func(what string, done, total int64) `json:"-"`
}

// say reports how far a download has got, and does nothing when nobody asked.
func (c Config) say(what string, done, total int64) {
	if c.Fetching != nil {
		c.Fetching(what, done, total)
	}
}

// LayoutModel divides a page into its parts and puts them in reading order.
type LayoutModel struct {
	// Name is where the model is fetched from, and Path is a file on this
	// machine. A path is used as given; a name is looked for in Dir first.
	Name string `json:"name"`
	Path string `json:"path"`

	// Labels are the parts the model knows, in the order of its class ids.
	Labels []string `json:"labels"`

	// Minimum is the score a part carries to be a part at all.
	Minimum float32 `json:"minimum"`
	// Overlap is how much of two parts may be common before the second is taken
	// to be the first found again.
	Overlap float64 `json:"overlap"`
	// Margin is what a part is widened by before it is read, because the first
	// letter of a line sits on the boundary the model drew.
	Margin int `json:"margin"`
}

// DetectModel finds the lines one part holds.
type DetectModel struct {
	Name string `json:"name"`
	Path string `json:"path"`

	// MaxSide is the longest side a part is read at.
	MaxSide int `json:"max_side"`
	// Expand is how many pixels a found line is widened by, in the image the
	// detector reads: the part scaled so that its longest side is MaxSide.
	Expand int `json:"expand"`
	// Minimum is the heat a pixel carries to be part of a line.
	Minimum float32 `json:"minimum"`
}

// RecogniserModel reads what a line says.
type RecogniserModel struct {
	Name string `json:"name"`
	Path string `json:"path"`

	// Dict is the model's characters, one to a line. Empty is the ordinary
	// case: the model carries them, and asking it is the only way to be sure.
	Dict string `json:"dict"`
	// Classes is how many characters the model knows and two more. Zero asks
	// the model. A number that disagrees with the model is refused.
	Classes int64 `json:"classes"`

	// Height is what a line is scaled to before it is read.
	Height int `json:"height"`
	// Sessions is how many lines are read at once.
	Sessions int `json:"sessions"`
}

// PageReading is how a page becomes an image, and how much of the machine one
// page may use.
type PageReading struct {
	// DPI is what a page is rendered at.
	DPI int `json:"dpi"`
	// Threads is how many threads one model may use.
	Threads int `json:"threads"`
}

// RegionKinds says what the parts of a page are for. A part the model names
// that Body does not carry is not read.
type RegionKinds struct {
	// Body carry what the document says.
	Body []string `json:"body"`
	// Head open a part of the document, outermost first: where a name stands in
	// the list is how deep the part it opens sits. Each of them is read only
	// where Body carries it too.
	Head []string `json:"head"`
}

// Defaults read a scanned page with the smallest models measured to be the best
// on one, and ask for the parts that carry prose.
func Defaults() Config {
	return Config{
		// Where each model is. Two come from one host and one from another,
		// because the export that carries its own alphabet is not the export the
		// first host publishes.
		Layout: LayoutModel{
			Name: "https://huggingface.co/PaddlePaddle/PP-DocLayoutV3_onnx/resolve/main/inference.onnx",
		},
		Detect: DetectModel{
			Name: "https://huggingface.co/PaddlePaddle/PP-OCRv5_mobile_det_onnx/resolve/main/inference.onnx",
		},
		Recognise: RecogniserModel{
			Name: "https://www.modelscope.cn/models/RapidAI/RapidOCR/resolve/v3.9.2/onnx/PP-OCRv6/rec/PP-OCRv6_rec_tiny.onnx",
		},
		Page: PageReading{DPI: 300, Threads: 4},

		// What a reading needs is fetched when it is wanted.
		Download: true,
	}
}

// The defaults for everything a settings file leaves out.
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

// labels are the parts PP-DocLayoutV3 knows, in the order of its class ids.
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

// The boundary the detector draws falls inside the letters by a share of the
// line's own height. One number serves a heading and a paragraph because a part
// is read scaled to MaxSide, which brings the two to nearly one size.
func (d DetectModel) expand() int {
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

func (r RecogniserModel) height() int {
	if r.Height <= 0 {
		return 48
	}
	return r.Height
}

func (r RecogniserModel) sessions() int {
	if r.Sessions <= 0 {
		return 4
	}
	return r.Sessions
}

func (p PageReading) dpi() int {
	if p.DPI <= 0 {
		return 300
	}
	return p.DPI
}

func (p PageReading) threads() int {
	if p.Threads <= 0 {
		return 4
	}
	return p.Threads
}

// body are the parts of a page that carry what the document says.
func (r RegionKinds) body() []string {
	if len(r.Body) > 0 {
		return r.Body
	}
	return []string{
		"text", "paragraph_title", "doc_title", "abstract", "content",
		"reference", "reference_content", "footnote", "figure_title",
	}
}

// head are the parts that open a part of the document, outermost first.
func (r RegionKinds) head() []string {
	if len(r.Head) > 0 {
		return r.Head
	}
	return []string{"doc_title", "paragraph_title"}
}

// paths are the files this run reads its models out of, and where they were
// found.
type paths struct {
	// engine is the ONNX Runtime this run loaded, and runtime is where it came
	// from.
	engine    *ort.Engine
	runtime   string
	layout    string
	detect    string
	recognise string
	from      string
}

// locate finds the runtime and the models.
//
// Three places are tried in order and each is a setting: a path written down,
// the folder the application was installed into, and what was downloaded. A
// path that is written down is used as given, and its absence is an error
// rather than a reason to look elsewhere — a person who said where a model is
// meant it.
func locate(ctx context.Context, cfg Config) (paths, error) {
	found := paths{from: "settings"}

	var err error
	if found.engine, found.runtime, err = library(ctx, cfg); err != nil {
		return paths{}, err
	}
	for _, one := range []struct {
		into             *string
		path, name, what string
	}{
		{&found.layout, cfg.Layout.Path, cfg.Layout.Name, "layout"},
		{&found.detect, cfg.Detect.Path, cfg.Detect.Name, "detect"},
		{&found.recognise, cfg.Recognise.Path, cfg.Recognise.Name, "recognise"},
	} {
		if *one.into, err = model(ctx, cfg, one.path, one.name, one.what); err != nil {
			return paths{}, err
		}
	}
	if cfg.Dir != "" {
		found.from = cfg.Dir
	}
	return found, nil
}

// model is where one model's file is.
//
// A path written down is used as given, and its absence is an error rather than
// a reason to look elsewhere: a person who said where a model is meant it. A
// name is looked for beside the application and then fetched.
func model(ctx context.Context, cfg Config, path, name, what string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("the %s model: %w", what, err)
		}
		return path, nil
	}
	if name == "" {
		return "", fmt.Errorf("no %s model: name one, or say where it is", what)
	}
	for _, at := range beside(cfg.Dir, filepath.Base(name)) {
		if _, err := os.Stat(at); err == nil {
			return at, nil
		}
	}
	if !address(name) {
		return "", fmt.Errorf("the %s model %q is not beside the application, and is not somewhere to fetch it from", what, name)
	}
	found, err := fetched(ctx, cfg, name, cfg.Download)
	if err != nil {
		return "", fmt.Errorf("the %s model: %w", what, err)
	}
	return found, nil
}

// beside is where a file may be: in the folder the settings name, and in the
// folder the application was installed into.
func beside(dir string, names ...string) []string {
	var out []string
	for _, name := range names {
		if dir != "" {
			out = append(out, filepath.Join(dir, name))
		}
		if self, err := os.Executable(); err == nil {
			out = append(out, filepath.Join(filepath.Dir(self), name))
			out = append(out, filepath.Join(filepath.Dir(self), "models", name))
		}
	}
	return out
}

// name is what a model is called, for the record kept beside what it produced.
func name(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// spaced is one line as the recogniser wrote it, with a run of space between
// words standing as one space.
func spaced(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
