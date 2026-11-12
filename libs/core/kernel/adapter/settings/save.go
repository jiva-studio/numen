package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
)

// Setting is one field of the file and its position in it, the first element being the owner:
// `app.appearance.theme` is `Setting{At: []string{"app", "appearance", "theme"}}`.
type Setting struct {
	At      []string
	Written any
}

// Save writes settings into the file, leaving every other byte as it was:
// section order, indentation and number formatting such as `0.30` survive.
// A field missing from the file is appended to its section; a missing file is
// written with only the named fields. A file that does not parse, or a result
// that cannot be read back into `into`, is not written.
func (f *Files) Save(path string, into Document, settings ...Setting) error {
	return f.run(path, func(path string) error {
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			raw = []byte("{}\n")
		} else if err != nil {
			return err
		}

		// The whole file has to parse before any of it is written, since what is
		// written is the file itself with one span of it replaced.
		var whole json.RawMessage
		if err := json.Unmarshal(raw, &whole); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		for _, setting := range settings {
			patched, err := applySetting(raw, setting)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			raw = patched
		}

		if err := readBack(raw, settings, into); err != nil {
			return fmt.Errorf("%s: %w: %w", path, settingsdomain.ErrNotASetting, err)
		}
		return writeFileAtomically(path, raw)
	})
}

func readBack(raw []byte, wrote []Setting, into Document) error {
	if err := json.Unmarshal(raw, into); err != nil {
		return err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	app, isObject := root[settingsdomain.AppOwner].(map[string]any)
	if !isObject {
		return nil
	}
	for _, declared := range readAppDeclaration() {
		steps := strings.Split(declared.Key, ".")
		label := settingsdomain.AppOwner + "." + declared.Key
		value, found := lookupPath(app, steps)
		if !found || !slices.ContainsFunc(wrote, func(one Setting) bool { return isFieldCovered(one.At, label) }) {
			continue
		}
		if err := checkValue(declared, nil, value); err != nil {
			return wrapAt(label, err)
		}
	}
	return nil
}

func isFieldCovered(pos []string, field string) bool {
	name := strings.Join(pos, ".")
	return field == name || strings.HasPrefix(field, name+".")
}

// errNotASection is a name on the way to a setting that the file holds as
// something other than an object.
var errNotASection = errors.New("not a JSON object; a setting cannot be written inside it")

// errRepeated is a name written twice in one section. The settings are read
// from the last of the two, and a span is replaced at the first.
var errRepeated = errors.New("appears more than once")

func applySetting(raw []byte, setting Setting) ([]byte, error) {
	if len(setting.At) == 0 {
		return nil, errors.New("the setting path is empty")
	}
	value, err := json.Marshal(setting.Written)
	if err != nil {
		return nil, err
	}
	return replaceValueAt(raw, setting.At, value, "")
}

func replaceValueAt(object []byte, pos []string, value []byte, outer string) ([]byte, error) {
	target, err := parseObjectShape(object)
	if err != nil {
		return nil, err
	}

	for _, one := range target.pairs {
		if one.key != pos[0] {
			continue
		}
		if len(pos) == 1 {
			return splice(object, one.from, one.to, value), nil
		}
		section, err := replaceValueAt(object[one.from:one.to], pos[1:], value, target.indent)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", one.key, err)
		}
		return splice(object, one.from, one.to, section), nil
	}

	return target.appendMember(object, pos, value, outer)
}

// member is one member of an object: its name, where that name is written, and
type member struct {
	key              string
	nameFrom, nameTo int
	from, to         int
}

// shape is an object as the file has it: its members in the order they were
// written, where another one would go, and the indentation they are laid out
// with.
type shape struct {
	pairs  []member
	last   int
	indent string
}

func parseObjectShape(object []byte) (shape, error) {
	decoder := json.NewDecoder(bytes.NewReader(object))
	open, err := decoder.Token()
	if err != nil {
		return shape{}, err
	}
	if brace, is := open.(json.Delim); !is || brace != '{' {
		return shape{}, errNotASection
	}

	target := shape{last: int(decoder.InputOffset())}
	for decoder.More() {
		one, err := readMember(decoder, object, target)
		if err != nil {
			return shape{}, err
		}
		target.pairs = append(target.pairs, one)
		target.last = one.to
	}
	if _, err := decoder.Token(); err != nil {
		return shape{}, err
	}
	if len(target.pairs) > 0 {
		target.indent = getIndent(object, target.pairs[0].from)
	}
	return target, nil
}

func readMember(decoder *json.Decoder, object []byte, target shape) (member, error) {
	name, err := decoder.Token()
	if err != nil {
		return member{}, err
	}
	key, is := name.(string)
	if !is {
		return member{}, errNotASection
	}
	if target.hasKey(key) {
		return member{}, fmt.Errorf("%q %w", key, errRepeated)
	}
	one := member{key: key}
	quoted, err := json.Marshal(key)
	if err != nil {
		return member{}, err
	}
	if head := object[:int(decoder.InputOffset())]; bytes.HasSuffix(head, quoted) {
		one.nameFrom, one.nameTo = len(head)-len(quoted), len(head)
	}

	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return member{}, err
	}
	// A decoded value is the exact bytes of it, so where it ends and how long it
	// is say where it began.
	end := int(decoder.InputOffset())
	one.from, one.to = end-len(value), end
	return one, nil
}

func (s shape) hasKey(key string) bool {
	for _, one := range s.pairs {
		if one.key == key {
			return true
		}
	}
	return false
}

func (s shape) appendMember(object []byte, pos []string, value []byte, outer string) ([]byte, error) {
	name, err := json.Marshal(pos[0])
	if err != nil {
		return nil, fmt.Errorf("encode the key %q: %w", pos[0], err)
	}

	if len(s.pairs) == 0 {
		// An empty object has no layout to follow, so the member is placed on its
		// own line, indented one step deeper than the key the object belongs to.
		indent := outer + "  "
		wrapped, err := wrapInSections(pos[1:], value, indent)
		if err != nil {
			return nil, err
		}
		body := string(name) + ": " + string(wrapped)
		return splice(object, s.last, s.last, []byte("\n"+indent+body+"\n"+outer)), nil
	}

	wrapped, err := wrapInSections(pos[1:], value, s.indent)
	if err != nil {
		return nil, err
	}
	body := string(name) + ": " + string(wrapped)
	if s.indent == "" {
		return splice(object, s.last, s.last, []byte(", "+body)), nil
	}
	return splice(object, s.last, s.last, []byte(",\n"+s.indent+body)), nil
}

func wrapInSections(pos []string, value []byte, indent string) ([]byte, error) {
	if len(pos) == 0 {
		return value, nil
	}
	name, err := json.Marshal(pos[0])
	if err != nil {
		return nil, fmt.Errorf("encode the key %q: %w", pos[0], err)
	}
	inside, err := wrapInSections(pos[1:], value, indent+"  ")
	if err != nil {
		return nil, err
	}
	if indent == "" {
		return []byte("{" + string(name) + ": " + string(inside) + "}"), nil
	}
	return []byte("{\n" + indent + "  " + string(name) + ": " + string(inside) + "\n" + indent + "}"), nil
}

func getIndent(object []byte, first int) string {
	head := object[:first]
	line := bytes.LastIndexByte(head, '\n')
	if line < 0 {
		return ""
	}
	rest := head[line+1:]
	return string(rest[:len(rest)-len(bytes.TrimLeft(rest, " \t"))])
}

func resolvePath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	// links is as many hops as a settings file is ever kept behind.
	const links = 32
	for range links {
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			return resolved
		}
		target, err := os.Readlink(abs)
		if err != nil {
			break
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(abs), target)
		}
		abs = filepath.Clean(target)
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return abs
	}
	return filepath.Join(dir, filepath.Base(abs))
}

// Files reads and writes settings files one at a time: each work reads a file and writes it back
// whole, and two at once would write over each other's change. A lock file beside the settings
// file keeps the writers of other processes apart too. Every writer of one file shares
// one Files.
type Files struct {
	mu sync.Mutex
}

// ErrNoFiles is returned by a settings read or write that was given no Files.
var ErrNoFiles = errors.New("settings: no Files value was given")

// NewFiles returns the access to settings files that one composition root holds.
func NewFiles() *Files { return &Files{} }

func (f *Files) run(path string, work func(path string) error) error {
	if f == nil {
		return ErrNoFiles
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	resolved := resolvePath(path)
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return err
	}
	return withLock(resolved, func() error { return work(resolved) })
}

func splice(raw []byte, from, to int, with []byte) []byte {
	if from < 0 || to < from || to > len(raw) {
		return raw
	}
	return slices.Concat(raw[:from], with, raw[to:])
}

func writeFileAtomically(path string, content []byte) error {
	return atomicfile.WriteFile(context.Background(), path, bytes.NewReader(content), 0o600)
}

func isUnwritable(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS)
}
