// Package engine renders a template text with text/template, over the data of a new note and a closed set of functions.
package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode"
)

const (
	// MaxOutput is the most bytes a template may write.
	MaxOutput = 1 << 20

	runLimit = 5 * time.Second
)

const (
	// KindText is a field that asks for a line of text.
	KindText = "text"
	// KindSelect is a field that asks for one of its options.
	KindSelect = "select"
)

var (
	// ErrTemplate is a template that does not parse or does not run.
	ErrTemplate = errors.New("invalid template")

	// ErrNotAnOption is a value given for a select field that is none of its options.
	ErrNotAnOption = errors.New("not one of the options")

	// ErrOutputTooLarge is a template that writes more than MaxOutput bytes.
	ErrOutputTooLarge = errors.New("the template writes too much text")

	// ErrTooSlow is a template that does not finish in time.
	ErrTooSlow = errors.New("the template runs too long")
)

// Funcs are functions a template may call besides the built-in ones.
type Funcs = template.FuncMap

// Data is what a template reads: the name, folder and id of the new note, the instant of the call and the values of its fields.
type Data struct {
	Title  string
	Folder string
	ID     string
	Now    time.Time
	Values map[string]string
}

// Result is the text after rendering.
type Result struct {
	Text string
}

// Field is a question a template asks before it is rendered.
type Field struct {
	Kind    string
	Label   string
	Options []string
	Default string
}

// Renderer renders templates. Its functions are the built-in ones and those given to New.
type Renderer struct {
	extra Funcs
}

// New returns a renderer whose templates may also call extra. A built-in function is not replaced by one of the same name.
func New(extra Funcs) Renderer { return Renderer{extra: extra} }

// ListFields lists the fields a template asks, in the order the first pass reaches them.
func (r Renderer) ListFields(ctx context.Context, name, text string, data Data) ([]Field, error) {
	return r.collect(ctx, name, text, data, new(ids))
}

func (r Renderer) collect(ctx context.Context, name, text string, data Data, issued *ids) ([]Field, error) {
	pass := newPass(data, nil, issued)
	if _, err := r.execute(ctx, name, text, pass, "missingkey=zero"); err != nil {
		return nil, err
	}
	return pass.fields, nil
}

// Render renders a template. A field with no value given takes its default; a text given empty stays empty.
func (r Renderer) Render(ctx context.Context, name, text string, data Data) (Result, error) {
	issued := new(ids)
	fields, err := r.collect(ctx, name, text, data, issued)
	if err != nil {
		return Result{}, err
	}
	values, err := resolve(fields, data.Values)
	if err != nil {
		return Result{}, err
	}
	data.Values = values
	out, err := r.execute(ctx, name, text, newPass(data, values, issued), "missingkey=error")
	if err != nil {
		return Result{}, err
	}
	return Result{Text: out}, nil
}

func resolve(fields []Field, given map[string]string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		value, isGiven := given[field.Label]
		if !isGiven {
			value = field.Default
		}
		switch {
		case field.Kind == KindText:
			value = keepOnOneLine(value)
		case !slices.Contains(field.Options, value):
			return nil, fmt.Errorf("%q is %w of %q: %s", value, ErrNotAnOption, field.Label, strings.Join(field.Options, ", "))
		}
		values[field.Label] = value
	}
	return values, nil
}

func keepOnOneLine(text string) string {
	clean := strings.Map(func(r rune) rune {
		if isBreak(r) || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, text)
	return strings.Join(strings.FieldsFunc(clean, isBreak), " ")
}

func isBreak(r rune) bool {
	return r == '\n' || r == '\r' || r == '\u0085' || r == '\u2028' || r == '\u2029'
}

func (r Renderer) execute(ctx context.Context, name, text string, p *pass, option string) (string, error) {
	funcs := getFuncs(p)
	for key, fn := range r.extra {
		if _, isBuiltin := funcs[key]; !isBuiltin {
			funcs[key] = fn
		}
	}
	parsed, err := template.New(name).Option(option).Funcs(funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrTemplate, err)
	}
	guard(parsed)
	return run(ctx, parsed, p)
}

type outcome struct {
	text string
	err  error
}

func run(ctx context.Context, parsed *template.Template, p *pass) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, runLimit)
	defer cancel()
	data := p.getData()
	done := make(chan outcome, 1)
	go func() {
		out := limitedWriter{check: ctx.Err, text: strings.Builder{}}
		err := parsed.Execute(&out, data)
		done <- outcome{text: out.text.String(), err: err}
	}()
	select {
	case finished := <-done:
		if finished.err == nil {
			return finished.text, nil
		}
		if ctx.Err() == nil {
			return "", fmt.Errorf("%w: %w", ErrTemplate, finished.err)
		}
	case <-ctx.Done():
	}
	p.stopped.Store(true)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("%w: %w", ErrTemplate, ErrTooSlow)
	}
	return "", ctx.Err()
}

type limitedWriter struct {
	check func() error
	text  strings.Builder
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if err := w.check(); err != nil {
		return 0, err
	}
	if w.text.Len()+len(data) > MaxOutput {
		return 0, ErrOutputTooLarge
	}
	return w.text.Write(data)
}
