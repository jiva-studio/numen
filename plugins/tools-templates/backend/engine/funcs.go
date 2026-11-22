package engine

import (
	"crypto/rand"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"
	"unicode"
)

var errOddDict = errors.New("dict takes pairs of a name and a value")

var errNoOptions = errors.New("select takes at least one option")

var errBadKey = errors.New("dict names are text")

// pass is one run of a template: the data it reads, the values its fields take, and the fields it met.
type pass struct {
	data       Data
	vars       map[string]string
	collecting bool
	fields     []Field
}

func newPass(data Data, values map[string]string) *pass {
	p := &pass{data: data, vars: values, collecting: values == nil, fields: nil}
	if p.collecting {
		p.vars = make(map[string]string)
	}
	return p
}

func (p *pass) getData() map[string]any {
	return map[string]any{"title": p.data.Title, "folder": p.data.Folder, "id": p.data.ID, "now": p.data.Now, "vars": p.vars}
}

func (p *pass) ask(kind, label string, options []string, fallback string) string {
	if !p.collecting {
		if value, isKnown := p.vars[label]; isKnown {
			return value
		}
		return fallback
	}
	if value, isKnown := p.vars[label]; isKnown {
		return value
	}
	p.vars[label] = fallback
	p.fields = append(p.fields, Field{Kind: kind, Label: label, Options: options, Default: fallback})
	return fallback
}

func (p *pass) instant(given []time.Time) time.Time {
	if len(given) == 0 {
		return p.data.Now
	}
	return given[len(given)-1]
}

func getFuncs(p *pass) Funcs {
	return template.FuncMap{
		"date": func(layout string, at ...time.Time) string {
			return formatTime(p.instant(at), layout, defaultDateFormat)
		},
		"time": func(layout string, at ...time.Time) string {
			return formatTime(p.instant(at), layout, defaultTimeFormat)
		},
		"addDays":   func(days int, at ...time.Time) time.Time { return p.instant(at).AddDate(0, 0, days) },
		"addMonths": func(months int, at ...time.Time) time.Time { return p.instant(at).AddDate(0, months, 0) },
		"weekday":   func(at ...time.Time) string { return p.instant(at).Weekday().String() },
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"slug":      slug,
		"title":     capitalize,
		"default":   fallback,
		"trunc":     truncate,
		"join":      join,
		"split":     func(separator, text string) []string { return strings.Split(text, separator) },
		"replace":   func(old, replacement, text string) string { return strings.ReplaceAll(text, old, replacement) },
		"trim":      strings.TrimSpace,
		"uuid":      newUUID,
		"cursor":    func() string { return cursorMarker },
		"dict":      dict,
		"list":      func(items ...any) []any { return items },
		"prompt": func(label string, defaults ...string) string {
			return p.ask(KindText, label, []string{}, first(defaults))
		},
		"select": func(label string, options ...string) (string, error) {
			if len(options) == 0 {
				return "", errNoOptions
			}
			return p.ask(KindSelect, label, options, options[0]), nil
		},
	}
}

const (
	defaultDateFormat = "YYYY-MM-DD"
	defaultTimeFormat = "HH:mm"
)

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func slug(text string) string {
	var out strings.Builder
	isBreak := false
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if isBreak && out.Len() > 0 {
				out.WriteByte('-')
			}
			isBreak = false
			out.WriteRune(r)
			continue
		}
		isBreak = true
	}
	return out.String()
}

func capitalize(text string) string {
	var out strings.Builder
	isStart := true
	for _, r := range text {
		if isStart {
			r = unicode.ToTitle(r)
		}
		isStart = unicode.IsSpace(r)
		out.WriteRune(r)
	}
	return out.String()
}

func fallback(replacement, value any) any {
	if isEmpty(value) {
		return replacement
	}
	return value
}

func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.String, reflect.Slice, reflect.Map, reflect.Array:
		return reflected.Len() == 0
	default:
		return reflected.IsZero()
	}
}

func truncate(length int, text string) string {
	runes := []rune(text)
	switch {
	case length < 0:
		return string(runes[max(len(runes)+length, 0):])
	case length < len(runes):
		return string(runes[:length])
	}
	return text
}

func join(separator string, items any) string {
	reflected := reflect.ValueOf(items)
	if reflected.Kind() != reflect.Slice {
		return fmt.Sprint(items)
	}
	parts := make([]string, reflected.Len())
	for i := range parts {
		parts[i] = fmt.Sprint(reflected.Index(i).Interface())
	}
	return strings.Join(parts, separator)
}

func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, errOddDict
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, isText := pairs[i].(string)
		if !isText {
			return nil, errBadKey
		}
		out[key] = pairs[i+1]
	}
	return out, nil
}

func newUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("make a uuid: %w", err)
	}
	raw[6] = raw[6]&0x0f | 0x40
	raw[8] = raw[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]), nil
}
