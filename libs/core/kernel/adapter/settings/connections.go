package settings

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// keyConnections is the key of the section of the file that lists the connections. It is no plugin
// id, because an id has a dot.
const keyConnections = "connections"

const jsonNull = "null"

// connectionEntry is a connection as the file lays it out. A secret value is never part of it.
type connectionEntry struct {
	ID       string         `json:"id"`
	TypeID   string         `json:"type"`
	Name     string         `json:"name"`
	Revision int            `json:"revision"`
	Fields   map[string]any `json:"fields"`
}

func (f *File) readConnections(raw json.RawMessage) {
	f.Connections = nil
	if string(raw) == jsonNull {
		return
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		f.AddNotice(fmt.Sprintf("numen.json: %s is not a list; its connections are ignored", keyConnections))
		return
	}
	for index, one := range entries {
		var entry connectionEntry
		if err := json.Unmarshal(one, &entry); err != nil {
			f.AddNotice(fmt.Sprintf("numen.json: %s[%d] has the wrong shape; the connection is ignored", keyConnections, index))
			continue
		}
		f.Connections = append(f.Connections, newConnection(entry))
	}
}

func newConnection(entry connectionEntry) connection.Connection {
	fields := entry.Fields
	if fields == nil {
		fields = map[string]any{}
	}
	return connection.Connection{
		ID: entry.ID, TypeID: entry.TypeID, Name: entry.Name, Revision: entry.Revision,
		Fields: fields,
	}
}

func newConnectionEntries(connections []connection.Connection) []connectionEntry {
	entries := make([]connectionEntry, 0, len(connections))
	for _, one := range connections {
		fields := one.Fields
		if fields == nil {
			fields = map[string]any{}
		}
		entries = append(entries, connectionEntry{
			ID: one.ID, TypeID: one.TypeID, Name: one.Name, Revision: one.Revision,
			Fields: fields,
		})
	}
	return entries
}

// DropInvalidConnections leaves out of force each connection that connection.FilterValid rejects, with a
// notice each. The file on disk is not touched.
func (f *File) DropInvalidConnections() {
	kept, rejections := connection.FilterValid(f.Connections)
	for _, one := range rejections {
		f.AddNotice(fmt.Sprintf("numen.json: connection %q: %s; the connection is ignored", one.ID, one.Reason))
	}
	f.Connections = kept
}

// ChangeConnections reads the connections section of the file, applies change and writes the section
// back, all under one hold of the lock. The rest of the file is kept byte for byte. Nothing is
// written where change fails or leaves the list as it was.
func (f *Files) ChangeConnections(path string, change func([]connection.Connection) ([]connection.Connection, error)) error {
	return f.run(path, func(path string) error {
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			raw = []byte("{}\n")
		} else if err != nil {
			return err
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(raw, &root); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var current File
		if section, found := root[keyConnections]; found {
			current.readConnections(section)
		}
		updated, err := change(slices.Clone(current.Connections))
		if err != nil {
			return err
		}
		if reflect.DeepEqual(newConnectionEntries(updated), newConnectionEntries(current.Connections)) {
			return nil
		}
		patched, err := applySetting(raw, Setting{At: []string{keyConnections}, Written: newConnectionEntries(updated)})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return writeFileAtomically(path, patched)
	})
}

// WriteConnectionsJSON is the connections section as the file lays it out.
func (f *File) WriteConnectionsJSON() (string, error) {
	content, err := json.Marshal(newConnectionEntries(f.Connections))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ConnectionRepository is the list of connections in the settings file.
type ConnectionRepository struct {
	files   *Files
	path    func() (string, error)
	read    func() (File, error)
	changed func()
}

// NewConnectionRepository returns the list of connections in the file at path. read answers the
// settings as they are in force, and changed is called after each write.
func NewConnectionRepository(files *Files, path func() (string, error), read func() (File, error), changed func()) *ConnectionRepository {
	return &ConnectionRepository{files: files, path: path, read: read, changed: changed}
}

// ListConnections returns the valid connections of the file in list order.
func (r *ConnectionRepository) ListConnections() ([]connection.Connection, error) {
	current, err := r.read()
	if err != nil {
		return nil, err
	}
	return current.Connections, nil
}

// ChangeConnections reads the connections section, applies change and writes it back under the lock
// of the settings file, and delivers the change to the readers of the setting-changed event.
func (r *ConnectionRepository) ChangeConnections(change func([]connection.Connection) ([]connection.Connection, error)) error {
	path, err := r.path()
	if err != nil {
		return err
	}
	if err := r.files.ChangeConnections(path, change); err != nil {
		return err
	}
	r.changed()
	return nil
}

// CreateConnectionID makes a random id for a connection: "c" and 8 lowercase hexadecimal digits.
func CreateConnectionID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a connection: %w", err)
	}
	return "c" + hex.EncodeToString(random[:]), nil
}
