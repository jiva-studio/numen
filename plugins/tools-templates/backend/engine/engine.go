// Package engine renders a template text with text/template, over the data of a new note and a closed set of functions.
package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"
)

const (
	cursorMarker = "\x00cursor\x00"

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

// Result is the text after rendering and the place of the first cursor marker.
// Cursor is an offset into Text in UTF-16 code units, -1 for none.
type Result struct {
	Text   string
	Cursor int
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

// Fields lists the fields a template asks, in the order the first pass reaches them.
func (r Renderer) Fields(name, text string, data Data) ([]Field, error) {
	pass := newPass(data, nil)
	if _, err := r.execute(name, text, pass, "missingkey=zero"); err != nil {
		return nil, err
	}
	return pass.fields, nil
}

// Render renders a template. A field with no value takes its default.
func (r Renderer) Render(name, text string, data Data) (Result, error) {
	fields, err := r.Fields(name, text, data)
	if err != nil {
		return Result{}, err
	}
	values, err := resolve(fields, data.Values)
	if err != nil {
		return Result{}, err
	}
	data.Values = values
	out, err := r.execute(name, text, newPass(data, values), "missingkey=error")
	if err != nil {
		return Result{}, err
	}
	return splitCursor(out), nil
}

func resolve(fields []Field, given map[string]string) (map[string]string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		value, isGiven := given[field.Label]
		switch {
		case !isGiven:
			value = field.Default
		case field.Kind == KindSelect && !slices.Contains(field.Options, value):
			return nil, fmt.Errorf("%q is %w of %q: %s", value, ErrNotAnOption, field.Label, strings.Join(field.Options, ", "))
		}
		values[field.Label] = value
	}
	return values, nil
}

func (r Renderer) execute(name, text string, p *pass, option string) (string, error) {
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
	var out strings.Builder
	if err := parsed.Execute(&out, p.getData()); err != nil {
		return "", fmt.Errorf("%w: %w", ErrTemplate, err)
	}
	return out.String(), nil
}

func splitCursor(text string) Result {
	before, after, found := strings.Cut(text, cursorMarker)
	if !found {
		return Result{Text: text, Cursor: -1}
	}
	return Result{Text: before + strings.ReplaceAll(after, cursorMarker, ""), Cursor: getUTF16Length(before)}
}

func getUTF16Length(text string) int {
	count := 0
	for _, r := range text {
		count++
		if r >= 0x10000 {
			count++
		}
	}
	return count
}
