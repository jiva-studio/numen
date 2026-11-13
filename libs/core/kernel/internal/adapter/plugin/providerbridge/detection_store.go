package providerbridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/atomicfile"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
)

// storedFile is the content of the store file of the detection: the answers by type id.
type storedFile struct {
	Answers map[string]storedAnswer `json:"answers"`
}

type storedAnswer struct {
	PluginVersion string             `json:"pluginVersion"`
	Connections   []storedConnection `json:"connections"`
}

type storedConnection struct {
	Name   string         `json:"name"`
	Fields map[string]any `json:"fields"`
}

func (d *Detection) load() {
	if d.isLoaded {
		return
	}
	d.isLoaded = true
	raw, err := os.ReadFile(d.storeFile)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	var stored storedFile
	if err == nil {
		err = json.Unmarshal(raw, &stored)
	}
	if err != nil {
		d.report(fmt.Errorf("read the detected connections: %w", err))
		return
	}
	for typeID, one := range stored.Answers {
		list := make([]connection.Connection, 0, len(one.Connections))
		for _, kept := range one.Connections {
			fields := kept.Fields
			if fields == nil {
				fields = map[string]any{}
			}
			list = append(list, connection.Connection{
				ID: connection.GetDetectedID(typeID, kept.Name), TypeID: typeID, Name: kept.Name, Fields: fields,
			})
		}
		d.answers[typeID] = answer{pluginVersion: one.PluginVersion, connections: list}
	}
}

func (d *Detection) writeStoreFile() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()
	path, stored := d.readStoredFile()
	if path == "" {
		return nil
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(path, bytes.NewReader(raw), 0o600)
}

func (d *Detection) readStoredFile() (string, storedFile) {
	d.mu.Lock()
	defer d.mu.Unlock()
	stored := storedFile{Answers: map[string]storedAnswer{}}
	for typeID, one := range d.answers {
		kept := storedAnswer{PluginVersion: one.pluginVersion, Connections: []storedConnection{}}
		for _, found := range one.connections {
			kept.Connections = append(kept.Connections, storedConnection{Name: found.Name, Fields: found.Fields})
		}
		stored.Answers[typeID] = kept
	}
	return d.storeFile, stored
}
