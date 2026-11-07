package settings

import (
	"bytes"
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

	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// Setting is one field of the file and its position in it, the first element being the owner:
// `app.appearance.theme` is `Setting{At: []string{"app", "appearance", "theme"}}`.
type Setting struct {
	At      []string
	Written any
}

// Save writes settings into the file, leaving every other byte of it as it was.
//
// The bytes of the named field's value are replaced in place and the rest of
// the file is preserved byte for byte, including section order, indentation,
// and number formatting such as `0.30`. A field missing from the file is
// appended to the end of the section it belongs to.
//
// A file that does not parse is not written, so the existing content is never
// lost to a failed save. A missing file is written containing only the named
// fields, and every field the file omits keeps its default.
//
// A result that cannot be read back into `into` is not written either: a
// value of the wrong shape is rejected. `into` defines the shape of the whole file, so a value
// written into any section is validated by reading the result back before it is saved.
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
			return fmt.Errorf("%s: %w: %w", path, settingsport.ErrNotASetting, err)
		}
		return writeFileAtomically(path, raw)
	})
}

// readBack returns an error where the file with the settings written is not one that reads back
// into the document, such as a value of the wrong shape for the section it is in, or where a
// setting of the shell that this call wrote holds a value its declaration does not accept. A value
// the file already held and this call did not write is left for the person to correct.
func readBack(raw []byte, wrote []Setting, into Document) error {
	if err := json.Unmarshal(raw, into); err != nil {
		return err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	app, isObject := root[OwnerApp].(map[string]any)
	if !isObject {
		return nil
	}
	for _, declared := range readAppDeclaration() {
		steps := strings.Split(declared.Key, ".")
		label := OwnerApp + "." + declared.Key
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

// isFieldCovered is whether a setting handed in at one name wrote the field at another: the field
// itself, or a field inside the section named.
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

// replaceValueAt replaces the value at a path through an object, and returns the
// object's bytes with that one span changed.
//
// what a section made here is laid out against.
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
	key string
	// nameFrom and nameTo are the name as written in the file, including quotes. Both
	// are zero for a name containing escapes, which is not renamed.
	nameFrom, nameTo int
	from, to         int
}

// shape is an object as the file has it: its members in the order they were
// written, where another one would go, and the indentation they are laid out
// with.
type shape struct {
	pairs []member
	// last is the end offset of the final member's value, where a comma and
	// another member are inserted. For an empty object it is the offset immediately after the brace.
	last int
	// indent is the whitespace before the first member's name. Empty for an
	// object written on one line and for one holding nothing.
	indent string
}

// parseObjectShape reads an object without disturbing it: what it holds, and where.
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

// readMember reads the next member of the object the decoder is in.
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

// hasKey is whether the object has a member of this name.
func (s shape) hasKey(key string) bool {
	for _, one := range s.pairs {
		if one.key == key {
			return true
		}
	}
	return false
}

// appendMember appends a member at the end of the object, using the object's
// existing layout. A new name goes last because the order of existing members is
// preserved.
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

// wrapInSections is the value a name is given, wrapped in the sections between
// it and the field it names. A section made for a one-line object is written on
// one line too.
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

// getIndent is the whitespace an object's members are laid out with, taken from
func getIndent(object []byte, first int) string {
	head := object[:first]
	line := bytes.LastIndexByte(head, '\n')
	if line < 0 {
		return ""
	}
	rest := head[line+1:]
	return string(rest[:len(rest)-len(bytes.TrimLeft(rest, " \t"))])
}

// resolvePath is where the bytes of the settings are, with every link on the
// way followed. A link is followed to its end whether or not anything is
// written there yet, and a path that resolves to nothing is its own answer.
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

// run hands the work the file the path leads to, with the folder that link is followed once and
// the rest of the way is the file itself.
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

// writeFileAtomically writes the file next to its target and renames it over the
// target, so a crash mid-write leaves the settings intact. The file mode is 0600.
//
// The path is the one run hands its work: the file itself, with every link
// on the way to it already followed.
func writeFileAtomically(path string, content []byte) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	// After a successful rename the temporary name no longer exists.
	defer func() {
		if removeErr := os.Remove(tmp.Name()); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

// syncDirectory flushes the folder the rename was recorded in. Flushing the file is
// what keeps its contents; flushing the folder is what keeps the rename.
//
// Not every filesystem lets a folder be opened for this, and the ones that
// refuse are the ones that did not need it.
func syncDirectory(dir string) error {
	folder, err := os.Open(dir)
	if err != nil {
		if isSyncUnsupported(err) {
			return nil
		}
		return err
	}
	if err := folder.Sync(); err != nil && !isSyncUnsupported(err) {
		return errors.Join(err, folder.Close())
	}
	return folder.Close()
}

// isSyncUnsupported is whether the filesystem refuses to flush a folder.
func isSyncUnsupported(err error) bool {
	return errors.Is(err, errors.ErrUnsupported) || errors.Is(err, fs.ErrPermission) ||
		errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP)
}

// isUnwritable is whether the file or its folder refuses writes. A read-only
// installation runs with the settings it reads, so this is not reported.
func isUnwritable(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.EROFS)
}
