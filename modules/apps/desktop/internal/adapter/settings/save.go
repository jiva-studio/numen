package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return replace(path, raw)
}

// errNotASection is a name on the way to a setting that the file holds as
// something other than an object.
var errNotASection = errors.New("a setting goes inside a section")

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

// pair is one member of an object: its name, and where its value sits in the
// bytes the object was read from.
type pair struct {
	key      string
	from, to int
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
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return shape{}, err
		}
		// A decoded value is the exact bytes of it, so where it ends and how
		// long it is say where it began.
		end := int(decoder.InputOffset())
		held.pairs = append(held.pairs, pair{key: key, from: end - len(value), to: end})
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

func spliced(raw []byte, from, to int, with []byte) []byte {
	patched := make([]byte, 0, len(raw)-(to-from)+len(with))
	patched = append(patched, raw[:from]...)
	patched = append(patched, with...)
	return append(patched, raw[to:]...)
}

// replace writes the file beside itself and renames it over the top, so a
// machine that dies mid-write leaves the settings whole. The mode is the
// person's alone: they type their service keys into this file.
func replace(path string, content []byte) error {
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
