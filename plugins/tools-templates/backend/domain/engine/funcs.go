package engine

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"text/template"
	"time"
	"unicode"
	"unicode/utf8"
)

var errOddDict = errors.New("dict takes pairs of a name and a value")

var errNoOptions = errors.New("select takes at least one option")

var errDisabled = errors.New("this function is not available in a template")

var errBadKey = errors.New("dict names are text")

var errFieldClash = errors.New("is asked differently elsewhere in the template")

var errFieldUnmet = errors.New("was not met when the template was first run, so it was not asked; ask it where the defaults reach it")

// ids are the identifiers a request hands out, shared by both passes so that the n-th call gets the same one in each.
type ids struct{ issued []string }

func (i *ids) get(n int) (string, error) {
	for len(i.issued) <= n {
		id, err := newUUID()
		if err != nil {
			return "", err
		}
		i.issued = append(i.issued, id)
	}
	return i.issued[n], nil
}

// pass is one run of a template: the data it reads, the values its fields take, and the fields it met.
type pass struct {
	data       Data
	vars       map[string]string
	collecting bool
	fields     []Field
	asked      map[string]Field
	ids        *ids
	called     int
	steps      int
	depth      int
	stopped    atomic.Bool
}

func newPass(data Data, values map[string]string, issued *ids) *pass {
	p := &pass{data: data, vars: values, collecting: values == nil, fields: nil, asked: make(map[string]Field), ids: issued, called: 0, steps: 0, depth: 0, stopped: atomic.Bool{}}
	if p.collecting {
		p.vars = make(map[string]string)
	}
	return p
}

func (p *pass) getData() map[string]any {
	return map[string]any{"title": p.data.Title, "folder": p.data.Folder, "id": p.data.ID, "now": p.data.Now, "vars": p.vars}
}

func (p *pass) newID() (string, error) {
	if p.stopped.Load() {
		return "", ErrTooSlow
	}
	p.called++
	return p.ids.get(p.called - 1)
}

func (p *pass) registerField(kind, label string, options []string, fallback string) (string, error) {
	if p.stopped.Load() {
		return "", ErrTooSlow
	}
	if !p.collecting {
		value, isKnown := p.vars[label]
		if !isKnown {
			return "", fmt.Errorf("field %q %w", label, errFieldUnmet)
		}
		return value, nil
	}
	if field, isAsked := p.asked[label]; isAsked {
		if field.Kind != kind || !slices.Equal(field.Options, options) {
			return "", fmt.Errorf("field %q %w", label, errFieldClash)
		}
		return p.vars[label], nil
	}
	field := Field{Kind: kind, Label: label, Options: options, Default: fallback}
	p.asked[label] = field
	p.vars[label] = fallback
	p.fields = append(p.fields, field)
	return fallback, nil
}

func (p *pass) getTimeOrNow(given []time.Time) time.Time {
	if len(given) == 0 {
		return p.data.Now
	}
	return given[len(given)-1]
}

func getFuncs(p *pass) Funcs {
	return template.FuncMap{
		"date": func(layout string, at ...time.Time) string {
			return formatTime(p.getTimeOrNow(at), layout, defaultDateFormat)
		},
		"time": func(layout string, at ...time.Time) string {
			return formatTime(p.getTimeOrNow(at), layout, defaultTimeFormat)
		},
		"addDays":   func(days int, at ...time.Time) time.Time { return p.getTimeOrNow(at).AddDate(0, 0, days) },
		"addMonths": func(months int, at ...time.Time) time.Time { return p.getTimeOrNow(at).AddDate(0, months, 0) },
		"weekday":   func(at ...time.Time) string { return p.getTimeOrNow(at).Weekday().String() },
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"slug":      buildSlug,
		"title":     formatCapitalized,
		"default":   selectFallback,
		"trunc":     truncate,
		"join":      join,
		"split":     func(separator, text string) []string { return strings.Split(text, separator) },
		"replace":   replace,
		"trim":      strings.TrimSpace,
		"call":      failDisabled,
		"html":      failDisabled,
		"js":        failDisabled,
		"printf":    failDisabled,
		"print":     failDisabled,
		"println":   failDisabled,
		"yaml":      quoteYAML,
		"uuid":      p.newID,
		tickName:    p.tick,
		enterName:   p.enter,
		leaveName:   p.leave,
		"dict":      buildDict,
		"list":      func(items ...any) []any { return items },
		"prompt": func(label string, defaults ...string) (string, error) {
			return p.registerField(KindText, label, []string{}, getFirstOrEmpty(defaults))
		},
		"select": func(label string, options ...string) (string, error) {
			if len(options) == 0 {
				return "", errNoOptions
			}
			return p.registerField(KindSelect, label, options, options[0])
		},
	}
}

const (
	defaultDateFormat = "YYYY-MM-DD"
	defaultTimeFormat = "HH:mm"
)

func failDisabled(...any) (string, error) { return "", errDisabled }

func getFirstOrEmpty(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func buildSlug(text string) string {
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

func formatCapitalized(text string) string {
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

func selectFallback(replacement, value any) any {
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

func replace(old, replacement, text string) (string, error) {
	if old != "" && len(text)+strings.Count(text, old)*(len(replacement)-len(old)) > MaxOutput {
		return "", ErrOutputTooLarge
	}
	if old == "" && len(text)+(utf8.RuneCountInString(text)+1)*len(replacement) > MaxOutput {
		return "", ErrOutputTooLarge
	}
	return strings.ReplaceAll(text, old, replacement), nil
}

func join(separator string, items any) (string, error) {
	reflected := reflect.ValueOf(items)
	if reflected.Kind() != reflect.Slice {
		return fmt.Sprint(items), nil
	}
	parts := make([]string, reflected.Len())
	size := max(len(parts)-1, 0) * len(separator)
	for i := range parts {
		parts[i] = fmt.Sprint(reflected.Index(i).Interface())
		if size += len(parts[i]); size > MaxOutput {
			return "", ErrOutputTooLarge
		}
	}
	return strings.Join(parts, separator), nil
}

func buildDict(pairs ...any) (map[string]any, error) {
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

func quoteYAML(text string) (string, error) {
	var out strings.Builder
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(text); err != nil {
		return "", fmt.Errorf("quote a value: %w", err)
	}
	return strings.TrimSuffix(out.String(), "\n"), nil
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
