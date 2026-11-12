package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// File is the file in which the user configures this installation. Its first level is the version
// `v` and the owners of settings: `app` holds the settings of the shell, and every other key is the
// id of a plugin with the settings of that plugin below it.
type File struct {
	Version     int
	Appearance  Appearance  `json:"appearance"`
	Titles      Titles      `json:"naming"`
	Performance Performance `json:"performance"`
	Plugins     map[string]map[string]any
	Connections []connection.Connection
	Notices     []string
}

// appObject is the object of the owner app as the file lays it out.
type appObject struct {
	Appearance  Appearance  `json:"appearance"`
	Titles      Titles      `json:"naming"`
	Performance Performance `json:"performance"`
}

// appSections are the sections of the owner app that a read fills in place, so that a section the
// file leaves out keeps the value it had.
type appSections struct {
	Appearance  *Appearance  `json:"appearance"`
	Titles      *Titles      `json:"naming"`
	Performance *Performance `json:"performance"`
}

const keyVersion = "v"

// GetVersion returns a pointer to the file format version.
func (f *File) GetVersion() *int { return &f.Version }

// GetAppearance returns a pointer to the appearance section.
func (f *File) GetAppearance() *Appearance { return &f.Appearance }

// AddNotice records one line about the file's contents, shown to the user.
func (f *File) AddNotice(notice string) { f.Notices = append(f.Notices, notice) }

// Sync is whether a note's title and its filename are kept as one name.
func (f *File) Sync() bool { return f.Titles.Sync() }

func (f *File) WriteJSON() (string, error) {
	content, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// MarshalJSON writes the version, the object of the owner app, the connections when there are any, and
// the object of each plugin by id.
func (f *File) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(`{`)
	members := []struct {
		key   string
		value any
	}{
		{keyVersion, f.Version},
		{settings.AppOwner, appObject{f.Appearance, f.Titles, f.Performance}},
	}
	if len(f.Connections) > 0 {
		members = append(members, struct {
			key   string
			value any
		}{keyConnections, newConnectionEntries(f.Connections)})
	}
	for _, id := range slices.Sorted(maps.Keys(f.Plugins)) {
		members = append(members, struct {
			key   string
			value any
		}{id, f.Plugins[id]})
	}
	for index, member := range members {
		if index > 0 {
			out.WriteString(`,`)
		}
		name, err := json.Marshal(member.key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(member.value)
		if err != nil {
			return nil, err
		}
		out.Write(name)
		out.WriteString(`:`)
		out.Write(value)
	}
	out.WriteString(`}`)
	return out.Bytes(), nil
}

// UnmarshalJSON reads the file over the values it already holds. A value of the shell that is
// undeclared, of the wrong type or outside its bounds is left out with a notice, and the default
// stays in force.
func (f *File) UnmarshalJSON(raw []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	hasInterfaceScale := false
	for _, key := range slices.Sorted(maps.Keys(root)) {
		switch key {
		case keyVersion:
			if err := json.Unmarshal(root[key], &f.Version); err != nil {
				return err
			}
		case settings.AppOwner:
			has, err := f.readApp(root[key])
			if err != nil {
				return err
			}
			hasInterfaceScale = has
		case keyConnections:
			f.readConnections(root[key])
		default:
			f.readPlugin(key, root[key])
		}
	}
	if !hasInterfaceScale {
		f.Appearance.InterfaceScale = getDesktopScale()
	}
	return nil
}

func (f *File) readApp(raw json.RawMessage) (hasInterfaceScale bool, err error) {
	app, isObject := decodeObject(raw)
	if !isObject {
		f.AddNotice(fmt.Sprintf("numen.json: %s is not a JSON object; its settings are ignored", settings.AppOwner))
		return false, nil
	}
	valid := map[string]any{}
	for _, declared := range readAppDeclaration() {
		path := strings.Split(declared.Key, ".")
		value, found := lookupPath(app, path)
		if !found || value == nil {
			continue
		}
		if err := checkValue(declared, nil, value); err != nil {
			f.AddNotice(fmt.Sprintf("numen.json: %q: %v; the value is ignored", settings.AppOwner+"."+declared.Key, err))
			continue
		}
		storePath(valid, path, value)
		hasInterfaceScale = hasInterfaceScale || declared.Key == keyInterfaceScale
	}
	for _, path := range listUndeclaredKeys(app) {
		f.AddNotice(fmt.Sprintf("numen.json: %q: no such setting; the value is ignored", settings.AppOwner+"."+path))
	}
	content, err := json.Marshal(valid)
	if err != nil {
		return false, err
	}
	sections := appSections{&f.Appearance, &f.Titles, &f.Performance}
	return hasInterfaceScale, json.Unmarshal(content, &sections)
}

func decodeObject(raw json.RawMessage) (map[string]any, bool) {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, false
	}
	return object, true
}

func (f *File) readPlugin(id string, raw json.RawMessage) {
	if !manifest.IsValidID(id) {
		f.AddNotice(fmt.Sprintf("numen.json: %s is not a valid plugin id; its settings are ignored", id))
		return
	}
	values, isObject := decodeObject(raw)
	if !isObject || values == nil {
		f.AddNotice(fmt.Sprintf("numen.json: %s is not a JSON object; its settings are ignored", id))
		return
	}
	if f.Plugins == nil {
		f.Plugins = map[string]map[string]any{}
	}
	f.Plugins[id] = values
}

func listUndeclaredKeys(app map[string]any) []string {
	declared := map[string]bool{}
	parents := map[string]bool{}
	for _, one := range readAppDeclaration() {
		declared[one.Key] = true
		steps := strings.Split(one.Key, ".")
		for end := 1; end < len(steps); end++ {
			parents[strings.Join(steps[:end], ".")] = true
		}
	}
	var found []string
	var walk func(prefix string, object map[string]any)
	walk = func(prefix string, object map[string]any) {
		for _, key := range slices.Sorted(maps.Keys(object)) {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			nested, isObject := object[key].(map[string]any)
			switch {
			case declared[path]:
			case parents[path] && isObject:
				walk(path, nested)
			default:
				found = append(found, path)
			}
		}
	}
	walk("", app)
	return found
}

func lookupPath(object map[string]any, path []string) (any, bool) {
	var current any = object
	for _, step := range path {
		next, isObject := current.(map[string]any)
		if !isObject {
			return nil, false
		}
		value, found := next[step]
		if !found {
			return nil, false
		}
		current = value
	}
	return current, true
}

func storePath(object map[string]any, path []string, value any) {
	for _, step := range path[:len(path)-1] {
		next, isObject := object[step].(map[string]any)
		if !isObject {
			next = map[string]any{}
			object[step] = next
		}
		object = next
	}
	object[path[len(path)-1]] = value
}

// DefaultFile is the file of an unconfigured installation. Every default is the one the
// declaration of the shell gives.
func DefaultFile() File {
	return File{
		Version:     DefaultVersion,
		Appearance:  DefaultAppearance(),
		Titles:      DefaultTitles(),
		Performance: DefaultPerformance(),
		Plugins:     map[string]map[string]any{},
	}
}
