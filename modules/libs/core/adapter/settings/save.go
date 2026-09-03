package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Setting is one field of the file and where in it that field sits:
// `appearance.theme` is `Setting{At: []string{"appearance", "theme"}}`.
type Setting struct {
	At    []string
	Value any
}

// Save writes settings into the file, leaving every other byte of it as it was.
//
// The bytes of the named field's value are replaced where they sit and the rest
// of the file is carried through untouched, so the order a person arranged
// their sections in, the way they indented them, and `0.30` written to two
// places all come back as they were. A field the file has not got is appended
// to the end of the section it belongs to.
//
// A file that does not parse is not written: what a person has in it is worth
// more than the setting being saved. A file that is not there is written
// holding the named fields alone, and every field a settings file leaves out
// keeps its default.
//
// A file the settings could not be read out of again is not written either: a
// value of the wrong shape, and a number past what its setting goes to, are
// refused where they are handed in.
func Save(path string, settings ...Setting) error {
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
		patched, err := set(raw, setting)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		raw = patched
	}

	if err := takes(raw, settings); err != nil {
		return fmt.Errorf("%s: %w: %w", path, port.ErrNotASetting, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return replace(path, raw)
}

// holds says what is wrong with the settings these bytes make, and nothing
// where they read out as settings this build can work with.
func holds(raw []byte) error {
	held := Defaults()
	if err := json.Unmarshal(raw, &held); err != nil {
		return err
	}
	return held.Appearance.Check()
}

// takes says what is wrong with the settings this call wrote, and nothing where
// each is a number its setting takes.
//
// A number outside its setting that the file already held is one the person
// typed and one they can still reach: what is refused is what was handed in.
func takes(raw []byte, wrote []Setting) error {
	held := Defaults()
	if err := json.Unmarshal(raw, &held); err != nil {
		return err
	}
	for _, outside := range held.Appearance.Outsides() {
		for _, setting := range wrote {
			if strings.Join(setting.At, ".") == outside.At {
				return outside
			}
		}
	}
	return nil
}

// rename gives one field of the file another name. Its value, its place among
// the fields around it and every other byte of the file stay as they are, so a
// file comes back from a rename the way its person wrote it, under one word.
//
// A file that has not got the field is left alone. A field whose new name the
// section already holds is left alone as well: one section holds one of a name.
func rename(path string, at []string, to string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// The whole file has to parse before any of it is written, since what is
	// written is the file itself with one span of it replaced.
	var whole json.RawMessage
	if err := json.Unmarshal(raw, &whole); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	renamed, done := named(raw, at, to)
	if !done {
		return nil
	}
	return replace(path, renamed)
}

// named hands back the object's bytes with one member's name changed, and
// whether it found the member to change.
func named(object []byte, at []string, to string) ([]byte, bool) {
	if len(at) == 0 {
		return object, false
	}
	held, err := members(object)
	if err != nil {
		return object, false
	}

	for _, one := range held.pairs {
		if one.key != at[0] {
			continue
		}
		if len(at) > 1 {
			section, done := named(object[one.from:one.to], at[1:], to)
			if !done {
				return object, false
			}
			return spliced(object, one.from, one.to, section), true
		}
		if one.nameTo == 0 || held.holds(to) {
			return object, false
		}
		name, err := json.Marshal(to)
		if err != nil {
			return object, false
		}
		return spliced(object, one.nameFrom, one.nameTo, name), true
	}

	return object, false
}

// errNotASection is a name on the way to a setting that the file holds as
// something other than an object.
var errNotASection = errors.New("a setting goes inside a section")

// errRepeated is a name written twice in one section. The settings are read
// from the last of them and a span is replaced at the first, so the file is
// left alone and the person is told which name to settle.
var errRepeated = errors.New("a section holds one of a name")

func set(raw []byte, setting Setting) ([]byte, error) {
	if len(setting.At) == 0 {
		return nil, errors.New("a setting with no name")
	}
	value, err := json.Marshal(setting.Value)
	if err != nil {
		return nil, err
	}
	return put(raw, setting.At, value, "")
}

// put replaces the value at a path through an object, and hands back the
// object's bytes with that one span changed.
//
// outer is the indentation of the line the object's own name sits on, which is
// what a section made here is laid out against.
func put(object []byte, at []string, value []byte, outer string) ([]byte, error) {
	held, err := members(object)
	if err != nil {
		return nil, err
	}

	for _, one := range held.pairs {
		if one.key != at[0] {
			continue
		}
		if len(at) == 1 {
			return spliced(object, one.from, one.to, value), nil
		}
		section, err := put(object[one.from:one.to], at[1:], value, held.indent)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", one.key, err)
		}
		return spliced(object, one.from, one.to, section), nil
	}

	return held.appending(object, at, value, outer), nil
}

// pair is one member of an object: its name, where that name is written, and
// where its value sits in the bytes the object was read from.
type pair struct {
	key string
	// nameFrom and nameTo are the name as the file has it, quotes and all. Both
	// are nought for a name written with escapes in it, which is a name this
	// stands well back from.
	nameFrom, nameTo int
	from, to         int
}

// shape is an object as the file has it: its members in the order they were
// written, where another one would go, and the indentation they are laid out
// with.
type shape struct {
	pairs []pair
	// last is where the final member's value ends, and where a comma and
	// another member go. An object holding nothing has it just past the brace.
	last int
	// indent is the whitespace before the first member's name. Empty for an
	// object written on one line and for one holding nothing.
	indent string
}

// members reads an object without disturbing it: what it holds, and where.
func members(object []byte) (shape, error) {
	decoder := json.NewDecoder(bytes.NewReader(object))
	open, err := decoder.Token()
	if err != nil {
		return shape{}, err
	}
	if brace, is := open.(json.Delim); !is || brace != '{' {
		return shape{}, errNotASection
	}

	held := shape{last: int(decoder.InputOffset())}
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			return shape{}, err
		}
		key, is := name.(string)
		if !is {
			return shape{}, errNotASection
		}
		// A name is read up to its closing quote, so the quoted name ends where
		// the decoder now stands. It is the file's own bytes only where they are
		// the plain quoting of it.
		if held.holds(key) {
			return shape{}, fmt.Errorf("%s: %w", key, errRepeated)
		}
		one := pair{key: key}
		quoted, err := json.Marshal(key)
		if err != nil {
			return shape{}, err
		}
		if head := object[:int(decoder.InputOffset())]; bytes.HasSuffix(head, quoted) {
			one.nameFrom, one.nameTo = len(head)-len(quoted), len(head)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return shape{}, err
		}
		// A decoded value is the exact bytes of it, so where it ends and how
		// long it is say where it began.
		end := int(decoder.InputOffset())
		one.from, one.to = end-len(value), end
		held.pairs = append(held.pairs, one)
		held.last = end
	}
	if _, err := decoder.Token(); err != nil {
		return shape{}, err
	}
	if len(held.pairs) > 0 {
		held.indent = indentOf(object, held.pairs[0].from)
	}
	return held, nil
}

// holds is whether the object has a member of this name.
func (s shape) holds(key string) bool {
	for _, one := range s.pairs {
		if one.key == key {
			return true
		}
	}
	return false
}

// appending puts a member at the end of the object, laid out the way the object
// already is. The end is where a name nobody has written yet belongs: the order
// of the rest is a person's.
func (s shape) appending(object []byte, at []string, value []byte, outer string) []byte {
	name, _ := json.Marshal(at[0])

	if len(s.pairs) == 0 {
		// An object holding nothing says nothing about how it is laid out, so
		// it is opened out one step past the name it hangs from.
		indent := outer + "  "
		body := string(name) + ": " + string(sectioned(at[1:], value, indent))
		return spliced(object, s.last, s.last, []byte("\n"+indent+body+"\n"+outer))
	}

	body := string(name) + ": " + string(sectioned(at[1:], value, s.indent))
	if s.indent == "" {
		return spliced(object, s.last, s.last, []byte(", "+body))
	}
	return spliced(object, s.last, s.last, []byte(",\n"+s.indent+body))
}

// sectioned is the value a name is given, wrapped in the sections between it
// and the field it names. A section made for a one-line object is written on
// one line too.
func sectioned(at []string, value []byte, indent string) []byte {
	if len(at) == 0 {
		return value
	}
	name, _ := json.Marshal(at[0])
	inside := sectioned(at[1:], value, indent+"  ")
	if indent == "" {
		return []byte("{" + string(name) + ": " + string(inside) + "}")
	}
	return []byte("{\n" + indent + "  " + string(name) + ": " + string(inside) + "\n" + indent + "}")
}

// indentOf is the whitespace an object's members are laid out with, taken from
// the line the first one sits on.
func indentOf(object []byte, first int) string {
	head := object[:first]
	line := bytes.LastIndexByte(head, '\n')
	if line < 0 {
		return ""
	}
	rest := head[line+1:]
	return string(rest[:len(rest)-len(bytes.TrimLeft(rest, " \t"))])
}

// resolved is where the bytes of the settings are, with every link on the way
// followed. A path that leads nowhere yet is resolved as far as its folder, and
// one that cannot be resolved at all is its own answer.
func resolved(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return abs
	}
	return filepath.Join(dir, filepath.Base(abs))
}

func spliced(raw []byte, from, to int, with []byte) []byte {
	patched := make([]byte, 0, len(raw)-(to-from)+len(with))
	patched = append(patched, raw[:from]...)
	patched = append(patched, with...)
	return append(patched, raw[to:]...)
}

// replace writes the file beside itself and renames it over the top, so a
// machine that dies mid-write leaves the settings whole. The mode is the
// person's alone: they type their service keys into this file.
//
// The rename lands on the file the path leads to, so a settings file kept in a
// dotfiles repository and linked into place is written where it is kept.
func replace(path string, content []byte) error {
	path = resolved(path)
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
